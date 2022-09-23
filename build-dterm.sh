#!/bin/bash

# clear
rm -f dterm
rm -rf statics
rm -rf frontend/dist/*

# build front
cd frontend/ && npm run build:local

# build server
cd ../backend || exit
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dterm dterm.go

# prepare
cd ../
mkdir statics
cp -r frontend/dist/* statics
cp backend/dterm .

# build image
docker build -t dterm-prod:v2.0.9 -f Dockerfile-dterm .