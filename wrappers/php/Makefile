.PHONY: install-header test install-dev-dependencies clean

EXPORTS_PATH ?= ../../exports
include $(EXPORTS_PATH)/common.mk

# Add phpunit to PATH
export PATH := $(abspath vendor/bin):$(PATH)

ifeq ($(COPY_LIB),1)
COPY_LIB_DIR ?= lib
endif
ifdef COPY_LIB_DIR
override COPY_LIB_DIR := $(COPY_LIB_DIR)/
endif

# Strip / replace elements confusing PHP's limited C parser: __SIZE_TYPE__, _Complex, extern "C"
# Also add FFI_LIB library name, see https://www.php.net/manual/en/ffi.load
install-header: .try-build-lib
	mkdir -p src/headers
	sed --null-data \
		-e 's/DO NOT EDIT/Modified for PHP/' \
		\
		-e 's/__SIZE_TYPE__/size_t/g' \
		-e 's/[^\n]*_Complex[^\n]*//g' \
		-e 's/#ifdef __cplusplus[^#]*#endif//g' \
		\
		-e 's/^/#define FFI_LIB "$(subst /,\/,$(COPY_LIB_DIR))$(LIB_FILE)"\n\n/' \
		\
		"$(EXPORTS_LIB_SUBFOLDER_PATH)/$(LIB_NAME).h" >src/headers/$(LIB_NAME)_php.h
ifdef COPY_LIB_DIR
	install "$(EXPORTS_LIB_SUBFOLDER_PATH)/$(LIB_FILE)" -Dt "$(COPY_LIB_DIR)"
endif

test: install-header install-dev-dependencies
	phpunit

composers = composer composer.phar ./composer ./composer.phar
# Find first composer from list above, default to 'composer'
find_composer = $(firstword $(foreach bin,$(composers),$(if $(shell command -v $(bin)),$(bin),)) composer)
# Lazy variable: https://blog.jgc.org/2016/07/lazy-gnu-make-variables.html
COMPOSER ?= $(eval COMPOSER := $(find_composer))$(COMPOSER)

# Try: composer, composer.phar, ./composer.phar, ./composer
# check-platform-reqs is needed because of config.platform in composer.json, see https://getcomposer.org/doc/06-config.md#platform
install-dev-dependencies:
	$(COMPOSER) install
	$(COMPOSER) check-platform-reqs

clean:
	rm -rf .phpunit* src/headers/*.h lib/*
ifeq ($(CLEAN_ALL),1)
	rm -rf vendor/
endif
