SHELL := /bin/bash
.DEFAULT_GOAL := help
.ONESHELL:

REGIONS := ap-northeast-1 ap-northeast-2 ap-northeast-3 ap-south-1 ap-southeast-1 ap-southeast-2 ca-central-1 eu-central-1 eu-north-1 eu-west-1 eu-west-2 eu-west-3 sa-east-1 us-east-1 us-east-2 us-west-1 us-west-2
S3_BUCKET_PREFIX ?= observeinc
VERSION ?= unreleased

define check_var
	@if [ -z "$($1)" ]; then
		echo >&2 "Please set the $1 variable";
		exit 2;
	fi
endef

SUBDIR = $(shell ls apps)

clean-aws-sam:
	rm -rf apps/*/.aws-sam/*

## help: shows this help message
help:
	@echo "Usage: make [target]"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## go-lint: runs linter for a given directory, specified via PACKAGE variable
go-lint:
	$(call check_var,PACKAGE)
	docker run --rm -v "`pwd`:/workspace:cached" -w "/workspace/$(PACKAGE)" golangci/golangci-lint:latest golangci-lint run

## go-lint-all: runs linter for all packages
go-lint-all:
	docker run --rm -v "`pwd`:/workspace:cached" -w "/workspace/." golangci/golangci-lint:latest golangci-lint run

## go-test: run go tests
go-test:
	go build ./...
	go test -v -race ./...

## sam-lint: validate and lint cloudformation templates
sam-lint:
	$(call check_var,APP)
	sam validate --lint --template apps/$(APP)/template.yaml

## sam-lint-all: validate and lint all cloudformation templates
sam-lint-all:
	for dir in $(SUBDIR); do
		APP=$$dir $(MAKE) sam-lint || exit 1;
	done

.PHONY: sam-build-all
## sam-build-all: build assets for all SAM applications across all regions
sam-build-all:
	@ for app in $(SUBDIR); do \
		for region in $(REGIONS); do \
			APP=$$app AWS_REGION=$$region $(MAKE) sam-build || exit 1; \
		done \
	done

## sam-build: build assets
sam-build:
	$(call check_var,APP)
	$(call check_var,AWS_REGION)
	cd apps/$(APP) && sam build --region $(AWS_REGION) --build-dir .aws-sam/build/$(AWS_REGION)


## sam-publish: publish serverless repo app
sam-publish: sam-package
	$(call check_var,AWS_REGION)
	sam publish \
	    --template-file apps/$(APP)/.aws-sam/build/$(AWS_REGION)/packaged.yaml \
	    --region $(AWS_REGION)

## sam-package-all: package all cloudformation templates and push assets to S3
sam-package-all:
	for dir in $(SUBDIR); do
		APP=$$dir $(MAKE) sam-package || exit 1;
	done

## sam-package: package cloudformation templates and push assets to S3
sam-package: sam-build
	$(call check_var,APP)
	$(call check_var,AWS_REGION)
	sam package \
	    --template-file apps/$(APP)/.aws-sam/build/$(AWS_REGION)/template.yaml \
	    --output-template-file apps/$(APP)/.aws-sam/build/$(AWS_REGION)/packaged.yaml \
	    --s3-bucket $(S3_BUCKET_PREFIX)-$(AWS_REGION) \
	    --s3-prefix apps/$(APP)/$(VERSION) \
	    --region $(AWS_REGION)
	aws s3 cp apps/$(APP)/.aws-sam/build/$(AWS_REGION)/packaged.yaml s3://$(S3_BUCKET_PREFIX)-$(AWS_REGION)/apps/$(APP)/$(VERSION)/

.PHONY: sam-package-all-regions
## sam-package-all-regions: Packages and uploads all SAM applications to S3 in multiple regions
sam-package-all-regions:
	@ for app in $(SUBDIR); do \
		for region in $(REGIONS); do \
			APP=$$app AWS_REGION=$$region $(MAKE) sam-package || exit 1; \
		done \
	done

## sam-publish-all: publish all apps
sam-publish-all:
	for dir in $(SUBDIR); do
		APP=$$dir $(MAKE) sam-publish || exit 1;
	done

build-App:
	$(call check_var,APP)
	$(call check_var,ARTIFACTS_DIR)
	GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o ./bootstrap cmd/$(APP)/main.go
	cp ./bootstrap $(ARTIFACTS_DIR)/.

build-Forwarder:
	APP=forwarder $(MAKE) build-App

.PHONY: help go-lint go-lint-all go-test sam-lint sam-lint-all sam-build sam-package sam-publish sam-package-all sam-publish-all build-App build-Forwarder

