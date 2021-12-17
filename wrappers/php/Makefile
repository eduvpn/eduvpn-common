.PHONY: install-header test install-dev-dependencies clean

ifneq (clean,$(MAKECMDGOALS))
GOOS      != go env GOHOSTOS
GOARCH    != go env GOHOSTARCH

export PATH := $(abspath vendor/bin):$(PATH)

ifeq (Windows_NT,$(OS))
export PATH := $(abspath ../../exports/$(GOOS)/$(GOARCH)):$(PATH)
else
export LD_LIBRARY_PATH := $(abspath ../../exports/$(GOOS)/$(GOARCH)):$(LD_LIBRARY_PATH)
endif
endif

install-header:
	$(MAKE) -C ../../exports
	mkdir -p src/headers
	sed --null-data \
		-e 's/DO NOT EDIT/Modified for PHP/' \
		-e 's/__SIZE_TYPE__/size_t/g' \
		-e 's/[^\n]*_Complex[^\n]*//g' \
		-e 's/#ifdef __cplusplus[^#]*#endif//g' \
		-e 's/^/#define FFI_LIB "eduvpn_verify"\n\n/' \
		"../../exports/$(GOOS)/$(GOARCH)/eduvpn_verify.h" > src/headers/eduvpn_verify_php.h

test: install-header install-dev-dependencies
	phpunit

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
	rm -rf vendor/ .phpunit* src/headers/*.h
