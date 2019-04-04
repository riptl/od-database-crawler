#!/usr/bin/env bash

appname="od-database-crawler"
outdir="dist/"
tag="${BUILD_ID}_$(date +%Y-%m-%d)"

rm -rf "./${outdir}"
mkdir build 2> /dev/null

name=${outdir}${appname}-${tag}-linux
GOOS="linux" GOARCH="amd64" go build -ldflags="-s -w" -o ${name}
gzip -f ${name}
echo ${name}

name=${outdir}${appname}-${tag}-mac
GOOS="darwin" GOARCH="amd64" go build -ldflags="-s -w" -o ${name}
gzip -f ${name}
echo ${name}

name=${outdir}${appname}-${tag}-freebsd
GOOS="freebsd" GOARCH="amd64" go build -ldflags="-s -w" -o ${name}
gzip -f ${name}
echo ${name}
