.PHONY: build copy-to clean

lib_prefix_linux   = lib
lib_prefix_windows =
lib_prefix_darwin  = lib

lib_suffix_linux   = .so
lib_suffix_windows = .dll
lib_suffix_darwin  = .dylib

GOOS      ?= $(shell go env GOHOSTOS)
GOARCH    ?= $(shell go env GOHOSTARCH)
LIB_PREFIX = $(lib_prefix_$(GOOS))
LIB_SUFFIX = $(lib_suffix_$(GOOS))

# Creates targets like 'linux/amd64/eduvpn_verify.so'
build: $(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify$(LIB_SUFFIX)

$(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify$(LIB_SUFFIX): exports.go ../verify.go
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ -buildmode=c-shared $<

copy-to: $(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify$(LIB_SUFFIX)
	install $< -Dt "$(COPY_TARGET)"

clean:
	rm -rf ../exports/*/
