# SRE-CHECKER

## Install

* Needs Golang and adding gobin in your path profile. *

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
      --rss-feed                  Running RSS Feed server.
      --tcp-host string           TCP server host to be track
      --tcp-port string           TCP server port to be track (default "80")
  -t, --timeout duration          Max timeout from service in seconds (default 30s)
      --unhealth-thresold int32   Consecutive failures (default 5)
```

### Example

`sre-checker --http-host tonto-http.nuvem.io --http-port 443 --tcp-host tonto.nuvem.io --tcp-port 3000 --health-thresold 5 --unhealth-thresold 5 --rss-feed --notify-email tonto@nuvem.io`
