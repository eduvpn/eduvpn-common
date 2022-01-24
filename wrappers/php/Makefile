.PHONY: install-header test install-dev-dependencies clean

EXPORTS_PATH ?= ../../exports
EXPORTS_LIB_PATH ?= $(EXPORTS_PATH)/lib

ifneq ($(MAKECMDGOALS),clean)
include $(EXPORTS_PATH)/platform.mk

# Add phpunit to PATH
export PATH := $(abspath vendor/bin):$(PATH)
endif

ifeq ($(COPY_LIB),1)
COPY_LIB_DIR = lib
endif

ifneq ($(COPY_LIB_DIR),)
COPY_LIB_DIR := $(COPY_LIB_DIR)/
endif

# Strip / replace elements confusing PHP's limited C parser: __SIZE_TYPE__, _Complex, extern "C"
# Also add FFI_LIB library name, see https://www.php.net/manual/en/ffi.load
install-header:
ifneq ($(EXPORTS_PATH),)
ifneq ($(wildcard $(EXPORTS_PATH)/Makefile),)
	$(MAKE) -C "$(EXPORTS_PATH)"
endif
endif
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
		"$(EXPORTS_LIB_PATH)/$(GOOS)/$(GOARCH)/$(LIB_NAME).h" > src/headers/$(LIB_NAME)_php.h
ifeq ($(COPY_LIB),1)
	install "$(EXPORTS_LIB_PATH)/$(GOOS)/$(GOARCH)/$(LIB_FILE)" -Dt "$(COPY_LIB_DIR)"
endif

test: install-header install-dev-dependencies
	phpunit

# Try: composer, composer.phar, ./composer.phar, ./composer
# check-platform-reqs is needed because of config.platform in composer.json, see https://getcomposer.org/doc/06-config.md#platform
install-dev-dependencies:
	if command -v composer; then \
		composer install && composer check-platform-reqs; \
	elif command -v composer.phar; then \
		composer.phar install && composer.phar check-platform-reqs; \
	else \
		./composer.phar install && ./composer.phar check-platform-reqs || \
		./composer install && ./composer check-platform-reqs; \
	fi

clean:
	rm -rf vendor/ .phpunit* src/headers/*.h lib/*
