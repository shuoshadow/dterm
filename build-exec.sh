#!/bin/bash

# clear
rm -f dterm-exec

# build server
cd backend || exit
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dterm-exec dterm_exec.go

# prepare
cd ../
cp backend/dterm-exec .

# build image
docker build -t dterm-exec:v1.1.0 -f Dockerfile-exec .
