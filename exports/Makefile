.PHONY: build clean

lib_suffix_linux   = .so
lib_suffix_windows = .dll
lib_suffix_darwin  = .dylib

GOOS      ?= $(shell go env GOHOSTOS)
GOARCH    ?= $(shell go env GOHOSTARCH)
LIB_SUFFIX = $(lib_suffix_$(GOOS))

# Creates targets like 'linux/amd64/eduvpn_verify.so'
build: $(GOOS)/$(GOARCH)/eduvpn_verify$(LIB_SUFFIX)

$(GOOS)/$(GOARCH)/eduvpn_verify$(LIB_SUFFIX): exports.go ../verify.go
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ -buildmode=c-shared $<

clean:
	rm -rf ../exports/*/
