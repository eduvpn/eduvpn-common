.PHONY: build clean

lib_suffix_linux   = .so
lib_suffix_windows = .dll
lib_suffix_darwin  = .dylib

OS         = $(shell go env GOHOSTOS)
ARCH       = $(shell go env GOHOSTARCH)
LIB_SUFFIX = $(lib_suffix_$(OS))

# Creates targets like 'linux/amd64/eduvpn_verify.so'
build: $(OS)/$(ARCH)/eduvpn_verify$(LIB_SUFFIX)

$(OS)/$(ARCH)/eduvpn_verify$(LIB_SUFFIX): exports.go ../verify.go
	CGO_ENABLED=1 GOOS=$(OS) GOARCH=$(ARCH) go build -o $@ -buildmode=c-shared $<

clean:
	rm -rf ../exports/*/
