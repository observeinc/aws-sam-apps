SHELL := /bin/bash
.DEFAULT_GOAL := help
.ONESHELL:

DBG_MAKEFILE ?=
ifeq ($(DBG_MAKEFILE),1)
    $(warning ***** starting Makefile for goal(s) "$(MAKECMDGOALS)")
    $(warning ***** $(shell date))
else
    # If we're not debugging the Makefile, don't echo recipes.
    MAKEFLAGS += -s
endif
# We don't need make's built-in rules.
MAKEFLAGS += --no-builtin-rules
# Be pedantic about undefined variables.
MAKEFLAGS += --warn-undefined-variables
.SUFFIXES:

-include variables.mk

LAMBDA_MAKEFILE = bin/$(OS)_$(ARCH)/Makefile

$(LAMBDA_MAKEFILE): $(GO_BUILD_DIRS)
	cp lambda.mk $@

.PHONY: clean
clean: # @HELP removes built binaries and temporary files.
clean: go-clean sam-clean

$(GO_BUILD_DIRS):
	mkdir -p $@

# The following structure defeats Go's (intentional) behavior to always touch
# result files, even if they have not changed.  This will still run `go` but
# will not trigger further work if nothing has actually changed.
GO_OUTBINS = $(foreach bin,$(GO_BINS),bin/$(OS)_$(ARCH)/$(bin))

go-build: # @HELP build Go binaries.
go-build: $(GO_OUTBINS)
	echo

# Each outbin target is just a facade for the respective stampfile target.
# This `eval` establishes the dependencies for each.
$(foreach outbin,$(GO_OUTBINS),$(eval  \
    $(outbin): .go/$(outbin).stamp  \
))

# This is the target definition for all outbins.
$(GO_OUTBINS):
	true

# Each stampfile target can reference an $(OUTBIN) variable.
$(foreach outbin,$(GO_OUTBINS),$(eval $(strip   \
    .go/$(outbin).stamp: OUTBIN = $(outbin)  \
)))

# This is the target definition for all stampfiles.
# This will build the binary under ./.go and update the real binary iff needed.
GO_STAMPS = $(foreach outbin,$(GO_OUTBINS),.go/$(outbin).stamp)
.PHONY: $(GO_STAMPS)
$(GO_STAMPS): go-build-bin
	echo -ne "binary: $(OUTBIN)  "
	if ! cmp -s .go/$(OUTBIN) $(OUTBIN); then  \
	    mv .go/$(OUTBIN) $(OUTBIN);            \
	    date >$@;                              \
	    echo;                                  \
	else                                       \
	    echo "(cached)";                       \
	fi

# This runs the actual `go build` which updates all binaries.
go-build-bin: | $(GO_BUILD_DIRS)
	echo "# building $(VERSION) for $(OS)/$(ARCH)"
	docker run                                                      \
	    -i                                                          \
	    --rm                                                        \
	    -u $$(id -u):$$(id -g)                                      \
	    -v $$(pwd):/src                                             \
	    -w /src                                                     \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin                    \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin/$(OS)_$(ARCH)      \
	    -v $$(pwd)/.go/cache:/.cache                                \
	    -v $$(pwd)/.go/pkg:/go/pkg                                  \
	    --env GOARCH=$(ARCH)                                        \
	    --env GOFLAGS="$(GOFLAGS) -mod=$(GO_MOD)"                   \
	    --env GOOS=$(OS)                                            \
	    $(GO_BUILD_IMAGE)                                           \
	    /bin/sh -c "                                                \
	        go install                                              \
	          -tags lambda.norpc                                    \
	          -ldflags \"-X $$(go list -m)/pkg/version.Version=$(VERSION)\"  \
	          ./...                                                 \
	    "

# This command is used for Orca scanning of our binaries 
docker-build-all-binaries-image: go-build-bin 
	@echo "### Building Docker image with ALL binaries for $(OS)/$(ARCH)"
	@$(eval IMAGE_NAME=$(or $(IMAGE_NAME),aws-sam-apps-all-binaries))

	# Ensure buildx builder exists and switch context correctly
	@if ! docker buildx inspect reproducible-builder >/dev/null 2>&1; then \
		echo "🔧 Creating buildx builder using docker-container driver..."; \
		docker buildx create --use --driver docker-container --name reproducible-builder; \
	else \
		echo "🔄 Using existing reproducible-builder"; \
		docker buildx use reproducible-builder; \
	fi

	docker buildx build \
		--build-arg OS=$(OS) \
		--build-arg ARCH=$(ARCH) \
		--build-arg VERSION=$(VERSION) \
		--output type=docker \
		--cache-from=type=local,src=.buildx-cache \
		--cache-to=type=local,dest=.buildx-cache \
		--tag $(IMAGE_NAME) \
		-f Dockerfile.all-binaries \
		.


go-clean: # @HELP clean Go temp files.
go-clean:
	test -d .go && chmod -R u+w .go || true
	rm -rf .go bin

go-test: # @HELP run Go unit tests.
go-test: | $(GO_BUILD_DIRS)
	docker run                                                  \
	    -i                                                      \
	    --rm                                                    \
	    -u $$(id -u):$$(id -g)                                  \
	    -v $$(pwd):/src                                         \
	    -w /src                                                 \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin                \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin/$(OS)_$(ARCH)  \
	    -v $$(pwd)/.go/cache:/.cache                            \
	    -v $$(pwd)/.go/pkg:/go/pkg                              \
	    --env GOFLAGS="$(GOFLAGS) -mod=$(GO_MOD)"               \
	    $(GO_BUILD_IMAGE)                                       \
	    /bin/sh -c "                                            \
	       go test ./...                                        \
	    "

go-lint: # @HELP lint Go workspace.
go-lint:
	docker run  --rm -v "$$(pwd):/workspace:cached" -w "/workspace/." golangci/golangci-lint:latest golangci-lint run --timeout 3m && echo "lint passed"

sam-clean: # @HELP remove SAM build directory.
sam-clean:
	rm -rf $(SAM_BUILD_DIR)

SAM_BUILD_TEMPLATES = $(foreach app,$(APPS), $(SAM_BUILD_DIR)/apps/$(app)/template.yaml)

$(foreach template,$(SAM_BUILD_TEMPLATE),$(eval  \
	$(template): apps/$(call get_app, $(template))/template.yaml \
))

$(SAM_BUILD_TEMPLATES): go-build $(LAMBDA_MAKEFILE)
	sam build \
	  -p \
	  -beta-features \
	  --template-file $(patsubst $(SAM_BUILD_DIR)/%,%,$@) \
	  --build-dir $(patsubst %template.yaml,%,$@) \
	  --config-file $(SAM_CONFIG_FILE) \
	  --config-env $(SAM_CONFIG_ENV)

SAM_PACKAGE_TARGETS = $(foreach app,$(APPS),sam-package-$(app))

.PHONY: $(SAM_PACKAGE_TARGETS)
# map each SAM_PACKAGE_TARGET to the corresponding SAM_PACKAGE_TEMPLATE for our current region
$(foreach target,$(SAM_PACKAGE_TARGETS),$(eval  \
    $(target): $(SAM_BUILD_DIR)/regions/$(AWS_REGION)/$(lastword $(subst -, , $(target))).yaml \
))

define check_var
       @[[ -n "$($1)" ]] || (echo >&2 "The environment variable '$1' is not set." && exit 2)
endef

define get_region
$(lastword $(subst /, ,$(basename $(dir $(1)))))
endef

define get_app
$(subst .yaml,,$(lastword $(subst /, ,$(1))))
endef

SAM_PACKAGE_DIRS = $(foreach region, $(AWS_REGIONS), $(SAM_BUILD_DIR)/regions/$(region))
SAM_PACKAGE_TEMPLATES = $(foreach dir,$(SAM_PACKAGE_DIRS), $(foreach app,$(APPS),$(dir)/$(app).yaml))

$(foreach template,$(SAM_PACKAGE_TEMPLATES),$(eval  \
	$(template): $(SAM_BUILD_DIR)/apps/$(call get_app, $(template))/template.yaml \
))

$(SAM_PACKAGE_DIRS):
	mkdir -p $@
	cp -r static/* $@

$(SAM_PACKAGE_TEMPLATES): | $(SAM_PACKAGE_DIRS)
	if [ ! -z "$(S3_BUCKET_PREFIX)" ]; then \
	  export FLAGS=" --s3-bucket $(S3_BUCKET_PREFIX)$(call get_region, $@)"; \
	else \
	  export FLAGS=" --resolve-s3"; \
    fi && \
	  sam package \
	    --template-file $(SAM_BUILD_DIR)/apps/$(call get_app, $@)/template.yaml \
	    --output-template-file $@                                               \
	    --region $(call get_region, $@)                                         \
	    --s3-prefix aws-sam-apps/$(VERSION)                                     \
	    --no-progressbar                                                        \
	    --config-file $(SAM_CONFIG_FILE)                                        \
	    --config-env $(SAM_CONFIG_ENV)                                          \
	    $${FLAGS}

SAM_PULL_REGION_TARGETS = $(foreach region,$(AWS_REGIONS),sam-pull-$(region))

$(foreach target,$(SAM_PULL_REGION_TARGETS),$(eval  \
	$(target): $(foreach app,$(APPS), $(SAM_BUILD_DIR)/regions/$(subst sam-pull-,,$(target))) \
))

.PHONY: $(SAM_PULL_REGION_TARGETS)
$(SAM_PULL_REGION_TARGETS): require_bucket_prefix
	# force ourselves to use the public URLs, verifying ACLs are correctly set
	cd $(SAM_BUILD_DIR)/regions/$(subst sam-pull-,,$@) && \
	for app in $(APPS); do \
	  curl -fs \
	    -O https://$(S3_BUCKET_PREFIX)$(subst sam-pull-,,$@).s3.$(subst sam-pull-,,$@).amazonaws.com/aws-sam-apps/$(VERSION)/$${app}.yaml \
	    -w "Pulled %{url_effective} status=%{http_code} size=%{size_download}\n" || exit 1; \
	done

SAM_PUSH_REGION_TARGETS = $(foreach region,$(AWS_REGIONS),sam-push-$(region))

$(foreach target,$(SAM_PUSH_REGION_TARGETS),$(eval  \
	$(target): $(foreach app,$(APPS), $(SAM_BUILD_DIR)/regions/$(subst sam-push-,,$(target))/$(app).yaml) \
))

require_bucket_prefix:
	$(call check_var,S3_BUCKET_PREFIX)

.PHONY: $(SAM_PUSH_REGION_TARGETS)
$(SAM_PUSH_REGION_TARGETS): require_bucket_prefix
	# ensure all previously pushed assets are public
	aws s3 cp \
	  --acl public-read \
	  --recursive \
	  s3://$(S3_BUCKET_PREFIX)$(subst sam-push-,,$@)/aws-sam-apps/$(VERSION)/ s3://$(S3_BUCKET_PREFIX)$(subst sam-push-,,$@)/aws-sam-apps/$(VERSION)/
	# push base manifests
	aws s3 cp \
	  --acl public-read \
	  --recursive \
	  $(SAM_BUILD_DIR)/regions/$(subst sam-push-,,$@)/ s3://$(S3_BUCKET_PREFIX)$(subst sam-push-,,$@)/aws-sam-apps/$(VERSION)/

SAM_VALIDATE_TARGETS = $(foreach app,$(APPS),sam-validate-$(app))

.PHONY: $(SAM_VALIDATE_TARGETS)
$(SAM_VALIDATE_TARGETS):
	yamllint apps/$(lastword $(subst -, ,$@))/template.yaml && \
	sam validate \
	--template apps/$(lastword $(subst -, ,$@))/template.yaml \
	--config-file $(SAM_CONFIG_FILE) \
	--config-env $(SAM_CONFIG_ENV)

TEST_INTEGRATION_TARGETS = $(foreach test,$(TF_TESTS),test-integration-$(test))

test-init:
	terraform -chdir=integration init

.PHONY: $(TEST_INTEGRATION_TARGETS)
$(TEST_INTEGRATION_TARGETS): test-init
	APP=$$(awk -F'"' '/^[[:space:]]*app[[:space:]]*=[[:space:]]*"/ {print $$2; exit}' integration/tests/$(lastword $(subst -, ,$@)).tftest.hcl); \
	if [ ! -z "$$APP" ]; then \
	  $(MAKE) sam-package-$$APP; \
	fi; \
	if [ "$(TF_TEST_DEBUG)" = "1" ]; then \
	  export CHECK_DEBUG_FILE=debug.sh; \
	fi && \
	  terraform -chdir=integration test -filter=tests/$(lastword $(subst -, ,$@)).tftest.hcl $(TF_TEST_ARGS);

TAG_REGION_TARGETS = $(foreach region,$(AWS_REGIONS),tag-$(region))

$(foreach target,$(TAG_REGION_TARGETS),$(eval  \
	$(target): sam-pull-$(subst tag-,,$(target)) \
))

$(TAG_REGION_TARGETS):
	$(call check_var,TAG)
	aws s3 sync \
	  --acl public-read \
	  --delete \
	  $(SAM_BUILD_DIR)/regions/$(subst tag-,,$@)/ s3://$(S3_BUCKET_PREFIX)$(subst tag-,,$@)/aws-sam-apps/$(TAG)/

.PHONY: sam-package
sam-package: # @HELP package all SAM templates.
sam-package: $(SAM_PACKAGE_TARGETS)

sam-package-%: # @HELP package specific SAM app (e.g sam-package-forwarder).

.PHONY: sam-pull
sam-pull: # @HELP pull SAM app manifests from remote URI to local build directory.
sam-pull: $(SAM_PULL_TARGETS)

sam-pull-%: # @HELP puall SAM app manifests for specific region (e.g sam-pull-us-west-2).

.PHONY: sam-push
sam-push: # @HELP package and push SAM assets to S3 to all regions.
sam-push: $(SAM_PUSH_REGION_TARGETS)

sam-push-%: # @HELP push all SAM apps to specific region (e.g sam-push-us-west-2)

.PHONY: sam-validate
sam-validate: # @HELP validate all SAM templates.
sam-validate: $(SAM_VALIDATE_TARGETS)

sam-validate-%: # @HELP validate specific SAM app (e.g. sam-validate-logwriter).

.PHONY: tag
tag: # @HELP pull SAM manifests for RELEASE_VERSION, and publish as RELEASE_TAG.
tag: $(TAG_REGION_TARGETS)

tag-%: # @HELP tag for specific region (e.g tag-us-west-2).


.PHONY: test-integration
test-integration: # @HELP run all integration tests.
test-integration: $(TEST_INTEGRATION_TARGETS)

test-integration-%: # @HELP run specific integration test (e.g. test-integration-stack).

.PHONY: version
version: # @HELP display version
version:
	echo "$(VERSION)"

.PHONY: parameters
parameters-%: # @HELP generate parameters list for documentation purposes.
	@echo "| Parameter       | Type    | Description |"
	@echo "|-----------------|---------|-------------|"
	@python3 -c 'import sys, yaml, json; y=yaml.safe_load(sys.stdin.read()); print(json.dumps(y))' < $(SAM_BUILD_DIR)/regions/$(AWS_REGION)/$(lastword $(subst -, , $@)).yaml | jq -r '.Parameters | to_entries[] | "| \(if .value.Default then "" else "**" end)`\(.key)`\(if .value.Default then "" else "**" end) | \(.value.Type) | \(.value.Description |  gsub("[\\n\\t]"; " ")) |"'


.PHONY: outputs
outputs-%: # @HELP generate outputs list for documentation purposes.
	@echo "| Output       |  Description |"
	@echo "|-----------------|-------------|"
	@python3 -c 'import sys, yaml, json; y=yaml.safe_load(sys.stdin.read()); print(json.dumps(y))' < $(SAM_BUILD_DIR)/regions/$(AWS_REGION)/$(lastword $(subst -, , $@)).yaml | jq -r '.Outputs | to_entries[] | "| \(.key) | \(.value.Description |  gsub("[\\n\\t]"; " ")) |"'

help: # @HELP displays this message.
help:
	echo "VARIABLES:"
	echo "  APPS          = $(APPS)"
	echo "  AWS_REGION    = $(AWS_REGION)"
	echo "  GO_BINS       = $(GO_BINS)"
	echo "  GO_BUILD_DIRS = $(GO_BUILD_DIRS)"
	echo "  TF_TESTS      = $(TF_TESTS)"
	echo "  VERSION       = $(VERSION)"
	echo
	echo "TARGETS:"
	grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) | cut -d':' -f2- \
	    | awk '                                   \
	        BEGIN {FS = ": *# *@HELP"};           \
	        { printf "  %-30s %s\n", $$1, $$2 };  \
	    '
