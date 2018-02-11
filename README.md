# go-gitea-webhook

Simple webhook receiver implementation for Gitea/Gogs. Based on [go-gitlab-webhook](https://github.com/soupdiver/go-gitlab-webhook).

Example `config.json`:

```json
{
  "logfile": "go-gitea-webhook.log",
  "address": "0.0.0.0",
  "port": 3344,
  "repositories": [
    {
      "secert": "verysecret123",
      "name": "user/repo",
      "commands": [
        "/home/user/command.sh"
      ]
    }
  ]
}
```
