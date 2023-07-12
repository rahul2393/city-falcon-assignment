# City Falcon Entity Microservice

## Overview

This repository contains the golang api server written on top of [fiber framework](https://github.com/gofiber/fiber)

## Prerequisites

### Software

- go version &gt;= 1.20 <https://golang.org/>
    - if you're on a sensible o/s: `$ pacman -S go go-tools`
    - if you're on Mac, you can install using Brew `brew install go`
        - You can install brew using `ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"`
    - for Windows/installers please visit [Golang Downloads](https://golang.org/dl/)
- &gt;= python 3.6
    - `pip install --user pypyr`

### Project root directory structure

Whilst we are not too prescriptive about the location of git repositories on your development machine, Golang does expect a certain structure so that it can store packages, binaries and find source code.  A sample root directory structure for all git repositories may look something like this:

```bash
/home/{user}/projects
```

Then set GOPATH environment variable to point to this location:

```bash
GOPATH="$HOME/projects"
export GOPATH
```

To make this more permanent, add the above lines into your *.bash_profile* file in your home directory.
Then, within this directory, create the following sub-directories:

```bash
/home/{user}/projects/bin
/home/{user}/projects/pkg
/home/{user}/projects/src
```

By default when the go tools are installed it creates *$HOME/go* which can be used, however, if you have many projects that support different languages then a *go* directory might be unintuitive as a project root directory.

#### Project directories within root directory structure

Golang projects need to be located within the *src* directory of the GOPATH location.  Given this restriction, it makes sense to follow general convention when it comes to defining the directories for git repository locations, organisations and projects within the *src* directory, as follows:

```bash
$GOPATH/src/{repo-location}/{organisation}/{project}
```

In this case of this repository, this equates to:

```bash
$GOPATH/src/github.com/rahul2393/city-falcon-assignment/
```

## Server configuration

All the server configuration can be managed via a environment values.

```bash
export LISTEN_ADDRESS_HTTP=8080
export DB_URL=postgres//{user}:{password}@{host}:5432/city_falcon?sslmode=disable
export LOG_QUERY=true
```

## Day-to-day build

```bash
go build -v -o bin/server . #creates a binary to run server
```

## APIs

### GET Slow Queries

Fetches the slow queries running on Postgres instance, API supports filtering on pg_stats_activity table coloumns

Example:
```bash
curl --location 'http://localhost:8080/slow-queries?filter=database_name!%3D%22%22'
```

### POST Entry

Creates an entry in the database

Example:
```bash
curl --location 'http://localhost:8080/entry' \
--header 'Content-Type: application/json' \
--data '{
    "version": 2
}'
```


### GET EntryBYID

List all the entries in the database

Example:
```bash
curl --location 'http://localhost:8080/entry/39a4fe61-4472-4205-99e0-96aa5258b1ab'
```

Note: To fetch deleted entries pass query param showDeleted=true

### GET Entries

List all the entries in the database

Example:
```bash
curl --location 'http://localhost:8080/entries'
```

### PUT Entry

Updates the entry identified by unique ID in the database

Example:
```bash
curl --location --request PUT 'http://localhost:8080/entry/39a4fe61-4472-4205-99e0-96aa5258b1ab?updateMask=version' \
--header 'Content-Type: application/json' \
--data '{
    "version": 3
}'
```

### Delete Entry

Soft deletes an entry from the database

Example:
```bash
curl --location --request DELETE 'http://localhost:8080/entry/39a4fe61-4472-4205-99e0-96aa5258b1ab'
```

### Server supports

1) LIST APIs pagination using `pageSize` and `pageOffset` query parameters.
2) GET SLOW Query supports filtering by SELECT, INSERT,UPDATE, DELETE using filter parameter
example query to fetch slow queries beginning with insert statements
```bash
curl --location 'http://localhost:8080/slow-queries?filter=query%3A%22INSERT%22'
```
3) In memory cache is used with TTL of 30 seconds and key API path.

## Architecture

    HTTP > handler/usecase
           handler/usecase > repository (Postgres)
           handler/usecase < repository (Postgres)
    HTTP < usecase