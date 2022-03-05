# 🔍 SRE-CHECKER 

`sre-checker` is a simple server status tracker that you can monitor TCP and HTTP services, with authentication, and get the health of applications. This services notify the status using email and you can enable an RSS Feed server to connect the status in Tracking Plugins.

## Install

*Needs Golang instaled and adding gobin in your path.*

Inside projects root directory execute.
```
go install .
```

## Usage
`sre-checker`

Setting up the .env token and login infomration like the example in `.env.example`

```
Usage:
  sre-checker [flags]

Flags:
      --check-interval duration   Check interval in seconds (default 5s)
      --config string             config file (default is $HOME/sre-checker.yaml)
      --health-thresold int32     Consecutive success (default 5)
  -h, --help                      help for sre-checker
      --http-host string          HTTP server host to be track
      --http-port string          HTTP server port to be track (default "80")
      --notify-email string       Email to get notifications
      --rss-feed                  Enable RSS Feed server.
      --rss-feed-host string      RSS Feed server host (default "0.0.0.0")
      --rss-feed-port string      RSS Feed server port (default "80")
      --tcp-host string           TCP server host to be track
      --tcp-port string           TCP server port to be track (default "80")
  -t, --timeout duration          Max timeout from service in seconds (default 30s)
      --unhealth-thresold int32   Consecutive failures (default 5)
```

When you enable `--rss-feed` you can reach a webserver in the address `http://<rss-feed-host>:<rss-feed-port>/rss` to connect in a RSS tracker.

### Example

`sre-checker --http-host tonto-http.nuvem.io --http-port 443 --tcp-host tonto.nuvem.io --tcp-port 3000 --health-thresold 5 --unhealth-thresold 5 --rss-feed --notify-email tonto@nuvem.io`

### Deploy for presentation

This repository has a github actions workflow for deploy this application on Heroku, you can use this workflow, but don't forget to set your own parameters in the file `.github/workflows/heroku.yml` and create needed secrets.

You can find the url for reach the RSS Feed in my case on the repository information. The application has to be stoped to avoid email SPAMs, if you want to run, get in touch.
