# This makefile is called by `sam package` when it is assembling template assets
# We do not compile the binaries in this Makefile. Instead, we rely on our
# Makefile to have already compiled everything prior to reaching this stage.
#
# This Makefile is copied into the Go build directory where it can be invoked by SAM.
# We do this because SAM will copy over the entire directory referenced in the
# CodeUri field. Relying on our root Makefile incurs a lot of delay due to the
# large number of files that would be copied over.
#
# SAM invokes this makefile with the target `build-<ResourceName>`. We name our
# lambda function resources according to the binary name, but capitalize them
# for consistency with the remainder of the SAM template. We must therefore
# make the copy case insensitive.
strip_and_lowercase = $(shell echo $(1) | sed 's/^build-//' | tr '[:upper:]' '[:lower:]')

build-%:
	cp $(call strip_and_lowercase,$@) $(ARTIFACTS_DIR)/bootstrap
