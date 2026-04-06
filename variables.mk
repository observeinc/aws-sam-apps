# Each directory under apps/* must contain a validate SAM template
APPS         := $(shell find apps/* -type d -maxdepth 0 -exec basename {} \;)

# This is our default region when not provided.
AWS_REGION   ?= us-west-2

# Regions to skip (set via environment to exclude broken/unavailable regions).
SKIP_REGIONS ?=

# List of regions supported by `make sam-push-*`.
AWS_REGIONS  := $(filter-out $(SKIP_REGIONS), \
                us-west-2      \
                us-west-1      \
                us-east-2      \
                us-east-1      \
                sa-east-1      \
                eu-west-3      \
                eu-west-2      \
                eu-west-1      \
                eu-north-1     \
                eu-central-1   \
                eu-central-2   \
                ca-central-1   \
                ca-west-1      \
                ap-southeast-2 \
                ap-southeast-1 \
                ap-south-1     \
                ap-northeast-3 \
                ap-northeast-2 \
                ap-northeast-1 \
                af-south-1     \
                me-south-1     \
                me-central-1   \
                il-central-1   \
                mx-central-1)

# Assume lambda functions are linux/arm64
# These variables must be defined before GO_BUILD_DIRS
OS              := $(if $(GOOS),$(GOOS),linux)
ARCH            := $(if $(GOARCH),$(GOARCH),arm64)

# Names of binaries to compile as lambda functions
GO_BINS         := forwarder subscriber metricsconfigurator pollerconfigurator
# Directories that we need created to build/test.
GO_BUILD_DIRS   := bin/$(OS)_$(ARCH)                   \
                .go/bin/$(OS)_$(ARCH)               \
                .go/cache                           \
                .go/pkg
# Build image to use for building lambda functions
GO_BUILD_IMAGE  ?= golang:1.25.7-alpine
# Which Go modules mode to use ("mod" or "vendor")
GO_MOD          ?= vendor
GOFLAGS         ?=

# Bucket prefix used when running `sam-push-*`. When omitted, a default
# bucket is auto-created as aws-sam-apps-${AWS_ACCOUNT_ID}-${REGION}.
S3_BUCKET_PREFIX ?=
SAM_BUILD_DIR    ?= .aws-sam/build

# Lambda binaries that need to be zipped and uploaded separately (not handled
# by SAM's native packaging because these templates use AWS::Lambda::Function).
LAMBDA_ZIP_DIR   := $(SAM_BUILD_DIR)/lambda-zips
LAMBDA_ZIP_BINS  := subscriber metricsconfigurator pollerconfigurator

# Per-app mapping: which Lambda ZIPs does each app need?
LAMBDA_ZIPS_logwriter     := subscriber
LAMBDA_ZIPS_metricstream  := metricsconfigurator
LAMBDA_ZIPS_externalrole  := pollerconfigurator
LAMBDA_ZIPS_stack         := subscriber metricsconfigurator pollerconfigurator
SAM_CONFIG_FILE  ?= $(shell pwd)/samconfig.yaml
SAM_CONFIG_ENV   ?= default

# List of tftests supported by `make test-integration-*`
TF_TESTS         ?= $(shell ls integration/tests | awk -F. '{print $$1}')
# Setting this flag to 1 will enable verbose logging and allow debugging of checks.
TF_TEST_DEBUG    ?= 0
TF_TEST_ARGS     ?=

# Tag is a symlink of sorts to an existing release.
# Our workflow sets RELEASE_TAG to match the release channel in semantic
# release. The default channel is '', which should be represented as `latest/`
TAG              := $(if $(RELEASE_TAG),$(RELEASE_TAG),latest)

# Version should only be overridden in CI. Cannot be empty.
VERSION          := $(if $(RELEASE_VERSION),$(RELEASE_VERSION),$(shell git describe --tags --always --dirty))

# When S3_BUCKET_PREFIX is not set, we auto-create a deterministic bucket.
# The account ID is lazily resolved only when needed.
AWS_ACCOUNT_ID   ?= $(shell aws sts get-caller-identity --query Account --output text 2>/dev/null)
RESOLVED_S3_BUCKET_PREFIX = $(if $(S3_BUCKET_PREFIX),$(S3_BUCKET_PREFIX),aws-sam-apps-$(AWS_ACCOUNT_ID)-)
