# gitable

[![Travis CI](https://travis-ci.org/jessfraz/gitable.svg?branch=master)](https://travis-ci.org/jessfraz/gitable)

Bot to automatically sync and update an [Airtable](https://airtable.com) sheet with 
GitHub pull request and issue data.


**NOTE:** Your airtable table must have the following fields: 
- `reference` **(single line text)**
- `title` **(single line text)** 
- `type` **(single select)**
- `state` **(single line text)**
- `author` **(single line text)**
- `labels` **(multiple select)**
- `comments` **(number)**
- `url` **(url)**
- `updated` **(date, include time)**
- `created` **(date, include  time)**
- `completed` **(date, include time)**
- `project` **(single line text)**

The only data you need to initialize **(if not running with `--autofill`)** 
is the `Reference` which is in the format
`{owner}/{repo}#{number}`.

It should look like the following:

![airtable.png](airtable.png)


## Installation

#### Airtable baseID and table name

 - [follow this guide](http://help.grow.com/connecting-your-data-internal-and-project-management/airtable/airtable-how-to-connect)

#### Binaries

- **darwin** [386](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-darwin-386) / [amd64](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-darwin-amd64)
- **freebsd** [386](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-freebsd-386) / [amd64](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-freebsd-amd64)
- **linux** [386](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-linux-386) / [amd64](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-linux-amd64) / [arm](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-linux-arm) / [arm64](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-linux-arm64)
- **solaris** [amd64](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-solaris-amd64)
- **windows** [386](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-windows-386) / [amd64](https://github.com/jessfraz/gitable/releases/download/v0.1.1/gitable-windows-amd64)

#### Via Go

```bash
$ go get github.com/jessfraz/gitable
```

#### Running with Docker

```console
$ docker run --restart always -d \
    -v /etc/localtime:/etc/localtime:ro \
    --name gitable \
    -e "GITHUB_TOKEN=59f6asdfasdfasdf0" \
    -e "AIRTABLE_APIKEY=ksdfsdf7" \
    -e "AIRTABLE_BASEID=appzxcvewrwtrewt4" \
    -e "AIRTABLE_TABLE=Current Open GitHub Pull Request and Issues" \
    r.j3ss.co/gitable --interval 1m
```

## Usage

```console
$ gitable -h
       _ _        _     _
  __ _(_) |_ __ _| |__ | | ___
 / _` | | __/ _` | '_ \| |/ _ \
| (_| | | || (_| | |_) | |  __/
 \__, |_|\__\__,_|_.__/|_|\___|
 |___/

 Bot to automatically sync and update an airtable sheet with
 GitHub pull request and issue data.
 Version: v0.1.1
 Build: 56b7db7

  -airtable-apikey string
        Airtable API Key (or env var AIRTABLE_APIKEY)
  -airtable-baseid string
        Airtable Base ID (or env var AIRTABLE_BASEID)
  -airtable-table string
        Airtable Table (or env var AIRTABLE_TABLE)
  -autofill
        autofill all pull requests and issues for a user [or orgs] to a table (defaults to current user unless --orgs is set)
  -d    run in debug mode
  -github-token string
        GitHub API token (or env var GITHUB_TOKEN)
  -interval duration
        update interval (ex. 5ms, 10s, 1m, 3h) (default 1m0s)
  -once
        run once and exit, do not run as a daemon
  -orgs value
        organizations to include (this option only applies to --autofill)
  -v    print version and exit (shorthand)
  -version
        print version and exit
```
