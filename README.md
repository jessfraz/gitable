# gitable

[![Travis CI](https://img.shields.io/travis/jessfraz/gitable.svg?style=for-the-badge)](https://travis-ci.org/jessfraz/gitable)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge)](https://godoc.org/github.com/jessfraz/gitable)

Bot to automatically sync and update an [Airtable](https://airtable.com) sheet with 
GitHub pull request and issue data.

**Table of Contents**

<!-- toc -->

- [Installation](#installation)
    + [Binaries](#binaries)
    + [Via Go](#via-go)
    + [Running with Docker](#running-with-docker)
- [Usage](#usage)
- [Airtable Setup](#airtable-setup)
    + [Using the API](#using-the-api)
    + [Format](#format)

<!-- tocstop -->

## Installation


#### Binaries

For installation instructions from binaries please visit the [Releases Page](https://github.com/jessfraz/gitable/releases).

#### Via Go

```console
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
gitable -  Bot to automatically sync and update an airtable sheet with GitHub pull request and issue data.

Usage: gitable <command>

Flags:

  --airtable-apikey  Airtable API Key (or env var AIRTABLE_APIKEY) (default: <none>)
  --airtable-baseid  Airtable Base ID (or env var AIRTABLE_BASEID) (default: <none>)
  --airtable-table   Airtable Table (or env var AIRTABLE_TABLE) (default: <none>)
  --autofill         autofill all pull requests and issues for a user [or orgs] to a table (defaults to current user unless --orgs is set) (default: false)
  -d, --debug        enable debug logging (default: false)
  --github-token     GitHub API token (or env var GITHUB_TOKEN)
  --interval         update interval (ex. 5ms, 10s, 1m, 3h) (default: 1m0s)
  --once             run once and exit, do not run as a daemon (default: false)
  --verbose-keys     include title data in keys
  --orgs             organizations to include (this option only applies to --autofill) (default: [])
  --watch-since      defines the starting point of the issues been watched (format: 2006-01-02T15:04:05Z). defaults to no filter (default: 2008-01-01T00:00:00Z)
  --watched          include the watched repositories (default: false)

Commands:

  version  Show the version information.
```

## Airtable Setup 

#### Using the API

[Follow this guide](https://help.grow.com/hc/en-us/articles/360015095834-Airtable).

#### Format

Your airtable table must have the following fields: 

- `Reference` **(single line text)**
- `Title` **(single line text)** 
- `Body` **(single line text)**
- `Type` **(single select)**
- `State` **(single select)**
- `Author` **(single line text)**
- `Labels` **(multiple select)**
- `Comments` **(number)**
- `URL` **(url)**
- `Updated` **(date, include time)**
- `Created` **(date, include  time)**
- `Completed` **(date, include time)**
- `Project` **(link to another sheet)**
- `Repository` **(single line text)**

The only data you need to initialize **(if not running with `--autofill`)** 
is the `Reference` which is in the format
`{owner}/{repo}#{number}`.

It should look like the following:

![airtable.png](airtable.png)
