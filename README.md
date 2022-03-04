# SRE-CHECKER

## Install

* Needs Golang and adding gobin in your path profile. *

Inside projects root directory execute.
```
go install .
```

## Usage
`sre-checker`

```
Usage:
  sre-checker [flags]

Flags:
      --auth string               Authentication token from server
      --check-interval duration   Check interval in seconds (default 5s)
      --config string             config file (default is $HOME/sre-checker.yaml)
      --health-thresold int32     Consecutive success (default 5)
  -h, --help                      help for sre-checker
      --http-host string          HTTP server host to be track
      --http-port string          HTTP server port to be track (default "80")
      --notify-email string       Email to get notifications
      --tcp-host string           TCP server host to be track
      --tcp-port string           TCP server port to be track (default "80")
  -t, --timeout duration          Max timeout from service in seconds (default 30s)
      --unhealth-thresold int32   Consecutive failures (default 5)
```
