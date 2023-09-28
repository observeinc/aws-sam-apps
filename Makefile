SHELL := /bin/bash

.PHONY: help
## help: shows this help message
help:
	@ echo "Usage: make [target]"
	@ sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: go-lint
## go-lint: runs linter for a given directory, specified via PACKAGE variable
go-lint: 
	@ if [ -z "$(PACKAGE)" ]; then echo >&2 please set directory via variable PACKAGE; exit 2; fi
	@ docker run  --rm -v "`pwd`:/workspace:cached" -w "/workspace/$(PACKAGE)" golangci/golangci-lint:latest golangci-lint run

.PHONY: go-lint-all
## go-lint-all: runs linter for all packages
go-lint-all: 
	@ docker run  --rm -v "`pwd`:/workspace:cached" -w "/workspace/." golangci/golangci-lint:latest golangci-lint run

.PHONY: go-test
## go-test: run go tests
go-test:
	@ go build ./...
	@ go test -v -race ./...

.PHONY: sam-lint
## sam-lint: validate and lint cloudformation templates
sam-lint:
	@ if [ -z "$(APP)" ]; then echo >&2 please set directory via variable APP; exit 2; fi
	@ sam validate --lint --template apps/$(APP)/template.yaml

SUBDIR = $(shell ls apps)
.PHONY: sam-lint
## sam-lint-all: validate and lint cloudformation templates
sam-lint-all:
	@ for dir in $(SUBDIR); do APP=$$dir $(MAKE) sam-lint || exit 1; done

.PHONY: sam-build
## sam-build: build assets
sam-build:
	@ if [ -z "$(APP)" ]; then echo >&2 please set directory via variable APP; exit 2; fi
	@ mkdir -p build/
	# build the lambda
	# requires Go to be installed
	@ sam build --template apps/$(APP)/template.yaml

.PHONY: sam-package
## sam-package: package cloudformation templates and push assets to S3
sam-package: sam-build
	# requires AWS credentials.
	# currently dynamically generates bucket. We will want to use a fixed set of buckets for our production artifacts.
	@ if [ -z "$(AWS_REGION)" ]; then echo >&2 please set AWS_REGION explicitly; exit 2; fi
	@ sam package --template apps/$(APP)/template.yaml --output-template-file build/$(APP).yaml --region $(AWS_REGION)

.PHONY: sam-publish
## sam-publish: publish serverless repo app
sam-publish: sam-package
	@ sam publish --template-file build/$(APP).yaml --region $(AWS_REGION)

.PHONY: sam-package-all
## sam-package-all: package all cloudformation templates and push assets to S3
sam-package-all:
	@ for dir in $(SUBDIR); do APP=$$dir $(MAKE) sam-package || exit 1; done
 
.PHONY: sam-publish-all
## sam-publish-all: publish all apps
sam-publish-all:
	@ for dir in $(SUBDIR); do APP=$$dir $(MAKE) sam-publish || exit 1; done
