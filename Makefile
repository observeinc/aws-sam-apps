SHELL := /bin/bash
.DEFAULT_GOAL := help
.ONESHELL:

VERSION ?= unreleased
# leave this undefined for the purposes of development
S3_BUCKET_PREFIX ?= 
AWS_REGION ?= $(shell aws configure get region)
SAM_BUILD_DIR ?= .aws-sam/build
SAM_CONFIG_FILE ?= $(shell pwd)/samconfig.yaml
SAM_CONFIG_ENV ?= default
BUILD_MAKEFILE_ENV_VARS = .make.env

DEBUG ?= 0

define check_var
	@[[ -n "$($1)" ]] || (echo >&2 "The environment variable '$1' is not set." && exit 2)
endef

SUBDIR = $(shell ls apps)

.PHONY: help go-lint go-lint-all go-test integration-test debug sam-validate sam-validate-all sam-build-all sam-build sam-publish sam-package-all sam-package release-all release sam-publish-all build-App build-Forwarder build-Subscriber clean-aws-sam

clean-aws-sam:
	rm -rf $(SAM_BUILD_DIR)

## help: Displays this help message listing all targets
help:
	@echo "Usage: make [target]"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## go-lint: Runs Go linter for a specified directory, set with PACKAGE variable
go-lint:
	$(call check_var,PACKAGE)
	docker run --rm -v "`pwd`:/workspace:cached" -w "/workspace/$(PACKAGE)" golangci/golangci-lint:latest golangci-lint run

## go-lint-all: Executes Go linter for all Go packages in the project
go-lint-all:
	docker run --rm -v "`pwd`:/workspace:cached" -w "/workspace/." golangci/golangci-lint:latest golangci-lint run

## go-test: Runs Go tests across all packages
go-test:
	go build ./...
	go test -v -race ./...

## integration-test: Executes integration tests, with optional debugging if DEBUG=1
integration-test:
	cd integration && terraform init && \
	if [ "$(DEBUG)" = "1" ]; then \
		CHECK_DEBUG_FILE=debug.sh terraform test $(TEST_ARGS); \
	else \
		terraform test $(TEST_ARGS); \
	fi

## sam-validate: Validates a specific CloudFormation template specified by APP variable
sam-validate:
	$(call check_var,APP)
	yamllint apps/$(APP)/template.yaml && \
	sam validate \
		--template apps/$(APP)/template.yaml \
		--config-file $(SAM_CONFIG_FILE) \
		--config-env $(SAM_CONFIG_ENV)

## sam-validate-all: Validates all CloudFormation templates in the project
sam-validate-all:
	@ for dir in $(SUBDIR); do \
		APP=$$dir $(MAKE) sam-validate || exit 1; \
	done

## sam-build-all: Builds assets for all SAM applications across specified regions
sam-build-all:
	@ for app in $(SUBDIR); do \
		for region in $(REGIONS); do \
			APP=$$app AWS_REGION=$$region $(MAKE) sam-build || exit 1; \
		done \
	done

## sam-build: Builds assets for a specific SAM application, specified by APP variable
sam-build:
	$(call check_var,APP)
	echo "VERSION=${VERSION}" > ${BUILD_MAKEFILE_ENV_VARS}
	sam build \
		--template-file apps/$(APP)/template.yaml \
		--build-dir $(SAM_BUILD_DIR)/$(APP)/$(AWS_REGION) \
		--config-file $(SAM_CONFIG_FILE) \
		--config-env $(SAM_CONFIG_ENV)

## sam-publish: Publishes a specific serverless repository application, after packaging
sam-publish: sam-package
	sam publish \
		--template-file $(SAM_BUILD_DIR)/$(APP)/$(AWS_REGION)/packaged.yaml \
		--region $(AWS_REGION) \
		--config-file $(SAM_CONFIG_FILE) \
		--config-env $(SAM_CONFIG_ENV)

## sam-package-all: Packages and pushes all CloudFormation templates to S3
sam-package-all:
	@ for dir in $(SUBDIR); do \
		APP=$$dir $(MAKE) sam-package || exit 1; \
	done

## sam-package: Packages a specific CloudFormation template and pushes assets to S3, specified by APP variable
sam-package: sam-build
	$(call check_var,APP)
	$(call check_var,VERSION)
	echo "Packaging for app: $(APP) in region: $(AWS_REGION)"
ifeq ($(S3_BUCKET_PREFIX),)
	sam package \
		--template-file $(SAM_BUILD_DIR)/$(APP)/$(AWS_REGION)/template.yaml \
		--output-template-file $(SAM_BUILD_DIR)/$(APP)/$(AWS_REGION)/packaged.yaml \
		--region $(AWS_REGION) \
		--resolve-s3 \
		--config-file $(SAM_CONFIG_FILE) \
		--config-env $(SAM_CONFIG_ENV)
else
	sam package \
		--template-file $(SAM_BUILD_DIR)/$(APP)/$(AWS_REGION)/template.yaml \
		--output-template-file $(SAM_BUILD_DIR)/$(APP)/$(AWS_REGION)/packaged.yaml \
		--region $(AWS_REGION) \
	    --s3-bucket $(S3_BUCKET_PREFIX)-$(AWS_REGION) \
	    --s3-prefix apps/$(APP)/$(VERSION) \
		--config-file $(SAM_CONFIG_FILE) \
		--config-env $(SAM_CONFIG_ENV)
endif

## release-all: Releases all applications, ensuring S3_BUCKET_PREFIX is set
release-all:
ifeq ($(S3_BUCKET_PREFIX),)
	$(error S3_BUCKET_PREFIX is empty. Cannot proceed with release-all.)
endif
	for dir in $(SUBDIR); do \
		APP=$$dir $(MAKE) release || exit 1; \
	done

## release: Packages, uploads, and sets ACL for a specific app, ensuring S3_BUCKET_PREFIX is set, specified by APP variable
release:
ifeq ($(S3_BUCKET_PREFIX),)
	$(error S3_BUCKET_PREFIX is empty. Cannot proceed with release.)
endif
	$(MAKE) sam-package
	@echo "Resetting assets to be public readable"
	aws s3 cp --acl public-read	--recursive s3://$(S3_BUCKET_PREFIX)-$(AWS_REGION)/apps/$(APP)/$(VERSION)/ s3://$(S3_BUCKET_PREFIX)-$(AWS_REGION)/apps/$(APP)/$(VERSION)/
	@echo "Copying stack definition"
	aws s3 cp --acl public-read $(SAM_BUILD_DIR)/$(APP)/$(AWS_REGION)/packaged.yaml s3://$(S3_BUCKET_PREFIX)-$(AWS_REGION)/apps/$(APP)/$(VERSION)/
ifeq ($(TAG),)
else
	aws s3 cp --acl public-read $(SAM_BUILD_DIR)/$(APP)/$(AWS_REGION)/packaged.yaml s3://$(S3_BUCKET_PREFIX)-$(AWS_REGION)/apps/$(APP)/$(TAG)/
endif

## sam-publish-all: Publishes all serverless applications
sam-publish-all:
	for dir in $(SUBDIR); do
		APP=$$dir $(MAKE) sam-publish || exit 1;
	done

build-App:
	$(call check_var,APP)
	$(call check_var,ARTIFACTS_DIR)
	GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -ldflags "-X $(shell go list -m)/version.Version=${VERSION}" -o ./bootstrap cmd/$(APP)/main.go
	cp ./bootstrap $(ARTIFACTS_DIR)/.

build-Forwarder:
	APP=forwarder $(MAKE) build-App

build-Subscriber:
	APP=subscriber $(MAKE) build-App

## parameters: generate doc table for cloudformation parameters
parameters:
	$(call check_var,APP)
	@echo "| Parameter       | Type    | Description |"
	@echo "|-----------------|---------|-------------|"
	@python3 -c 'import sys, yaml, json; y=yaml.safe_load(sys.stdin.read()); print(json.dumps(y))' < $(SAM_BUILD_DIR)/$(APP)/$(AWS_REGION)/template.yaml | jq -r '.Parameters | to_entries[] | "| \(if .value.Default then "" else "**" end)`\(.key)`\(if .value.Default then "" else "**" end) | \(.value.Type) | \(.value.Description |  gsub("[\\n\\t]"; " ")) |"'

## static-validate: validate any static assets
static-validate:
	@ yamllint --no-warnings static/

## static-upload: upload static assets
static-upload: static-validate
	$(call check_var,S3_BUCKET_PREFIX)
	aws s3 sync static s3://$(S3_BUCKET_PREFIX)/ --acl public-read --metadata Version=$(VERSION)

.PHONY: help go-lint go-lint-all go-test sam-validate sam-validate-all sam-build sam-package sam-publish sam-package-all sam-publish-all build-App build-Forwarder
