#!/bin/bash

export GO15VENDOREXPERIMENT="1";
#export GOPATH="/d1/gopath";
export GOPATH="$HOME/gopath";

env GOARM=6 GOARCH=arm GOOS=linux go build .