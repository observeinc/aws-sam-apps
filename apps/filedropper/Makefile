SHELL := /bin/bash

build-FiledropperFunction:
	GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o ./bootstrap ../../cmd/filedropper/main.go
	cp ./bootstrap $(ARTIFACTS_DIR)/.

