#!/bin/bash

cd $(dirname $(realpath $0))

binary=yandex-radio

GOOS=linux GOARCH=amd64 go build -o ${binary}.linux
GOOS=darwin GOARCH=amd64 go build -o ${binary}.darwin
GOOS=windows GOARCH=amd64 go build -o ${binary}.windows.exe

go install
