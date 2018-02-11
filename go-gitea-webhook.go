// Based on: https://github.com/soupdiver/go-gitlab-webhook
// Gitea SDK: https://godoc.org/code.gitea.io/sdk/gitea
// Gitea webhooks: https://docs.gitea.io/en-us/webhooks

package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	api "code.gitea.io/sdk/gitea"
)

//ConfigRepository represents a repository from the config file
type ConfigRepository struct {
	Secret   string
	Name     string
	Commands []string
}

//Config represents the config file
type Config struct {
	Logfile      string
	Address      string
	Port         int64
	Repositories []ConfigRepository
}

func panicIf(err error, what ...string) {
	if err != nil {
		if len(what) == 0 {
			panic(err)
		}

		panic(errors.New(err.Error() + (" " + what[0])))
	}
}

var config Config
var configFile string

func main() {
	args := os.Args

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP)

	go func() {
		<-sigc
		config = loadConfig(configFile)
		log.Println("config reloaded")
	}()

	//if we have a "real" argument we take this as conf path to the config file
	if len(args) > 1 {
		configFile = args[1]
	} else {
		configFile = "config.json"
	}

	//load config
	config = loadConfig(configFile)

	//open log file
	writer, err := os.OpenFile(config.Logfile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	panicIf(err)

	//close logfile on exit
	defer func() {
		writer.Close()
	}()

	//setting logging output
	log.SetOutput(writer)

	//setting handler
	http.HandleFunc("/", hookHandler)

	address := config.Address + ":" + strconv.FormatInt(config.Port, 10)

	log.Println("Listening on " + address)

	//starting server
	err = http.ListenAndServe(address, nil)
	if err != nil {
		log.Println(err)
	}
}

func loadConfig(configFile string) Config {
	var file, err = os.Open(configFile)
	panicIf(err)

	// close file on exit and check for its returned error
	defer func() {
		panicIf(file.Close())
	}()

	buffer := make([]byte, 1024)

	count, err := file.Read(buffer)
	panicIf(err)

	err = json.Unmarshal(buffer[:count], &config)
	panicIf(err)

	return config
}

func hookHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	log.Printf("RemoteAddr: %v\n", r.RemoteAddr)

	var hook api.PushPayload

	event := r.Header.Get("X-Gogs-Event")
	if event != "push" {
		log.Printf("receive unknown event \"%s\"\n", event)
		return
	}

	//read request body
	var data, err = ioutil.ReadAll(r.Body)
	panicIf(err, "while reading request")

	//unmarshal request body
	err = json.Unmarshal(data, &hook)
	panicIf(err, fmt.Sprintf("while unmarshaling request \"%s\"", b64.StdEncoding.EncodeToString(data)))

	log.Printf("received webhook on %s", hook.Repo.FullName)

	//find matching config for repository name
	for _, repo := range config.Repositories {
		if repo.Name != hook.Repo.FullName {
			continue
		}

		if repo.Secret != hook.Secret {
			log.Printf("secret mismatch for repo %s\n", repo.Name)
			continue
		}

		//execute commands for repository
		for _, cmd := range repo.Commands {
			var command = exec.Command(cmd)
			out, err := command.Output()
			if err != nil {
				log.Println(err)
			} else {
				log.Println("Executed: " + cmd)
				log.Println("Output: " + string(out))
			}
		}
	}
}
