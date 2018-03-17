# gitable

[![Travis CI](https://travis-ci.org//home/jessie/.go/src/github.com/jessfraz/gitable.svg?branch=master)](https://travis-ci.org//home/jessie/.go/src/github.com/jessfraz/gitable)

Bot to update an airtable sheet with GitHub pull request or issue data.

## Installation

#### Binaries

- **darwin** [386](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-darwin-386) / [amd64](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-darwin-amd64)
- **freebsd** [386](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-freebsd-386) / [amd64](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-freebsd-amd64)
- **linux** [386](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-linux-386) / [amd64](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-linux-amd64) / [arm](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-linux-arm) / [arm64](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-linux-arm64)
- **solaris** [amd64](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-solaris-amd64)
- **windows** [386](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-windows-386) / [amd64](https://github.com//home/jessie/.go/src/github.com/jessfraz/gitable/releases/download/v0.0.0/gitable-windows-amd64)

#### Via Go

```bash
$ go get github.com/jessfraz/gitable
```

## Usage

```console
       _ _        _     _
  __ _(_) |_ __ _| |__ | | ___
 / _` | | __/ _` | '_ \| |/ _ \
| (_| | | || (_| | |_) | |  __/
 \__, |_|\__\__,_|_.__/|_|\___|
 |___/

 Bot to update an airtable sheet with GitHub pull request or issue data.
 Version: v0.0.0
 Build: 6a3dee6

  -airtable-apikey string
        Airtable API Key (or env var AIRTABLE_APIKEY)
  -airtable-baseid string
        Airtable Base ID (or env var AIRTABLE_BASEID)
  -airtable-table string
        Airtable Table (or env var AIRTABLE_TABLE)
  -d    run in debug mode
  -github-token string
        GitHub API token (or env var GITHUB_TOKEN)
  -interval string
        update interval (ex. 5ms, 10s, 1m, 3h) (default "1m")
  -v    print version and exit (shorthand)
  -version
        print version and exit
```
