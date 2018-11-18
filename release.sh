#!/usr/bin/env bash

appname="od-database-crawler"
tag=$1
[ -z "$tag" ] && echo "Usage: build <version>" && exit 1

name=${appname}-${tag}-windows.exe
GOOS="windows" GOARCH="amd64" go build -ldflags="-s -w" -o $name
gzip -f $name
echo $name

name=${appname}-${tag}-linux
GOOS="linux" GOARCH="amd64" go build -ldflags="-s -w" -o $name
gzip -f $name
echo $name

name=${appname}-${tag}-mac
GOOS="darwin" GOARCH="amd64" go build -ldflags="-s -w" -o $name
gzip -f $name
echo $name
