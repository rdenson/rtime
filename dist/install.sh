#!/bin/bash

GOOS=darwin \
GOARCH=amd64 \
go build \
  -ldflags="-s -w -X 'github.com/rdenson/rtime/cmd.ToolVersion=$(git rev-parse --short HEAD)'"

./rtime completion bash > /usr/local/etc/bash_completion.d/rtime
