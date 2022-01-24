.PHONY: build copy-to clean

include platform.mk

ifeq ($(LIB_SUFFIX),.so)
# Add SONAME as cgo does not currently do this. Mostly for Android, see https://stackoverflow.com/a/48291044
export CGO_LDFLAGS := $(CGO_LDFLAGS) -Wl,-soname,$(LIB_FILE)
endif

# Creates targets like 'lib/linux/amd64/libeduvpn_common.so'
build: lib/$(GOOS)/$(GOARCH)/$(LIB_FILE)

# Build shared library and remove lib prefix (if any) from header name
# GOOS and GOARCH envvars are set by platform.mk
# This extra target prevents unnecessary rebuild
lib/$(GOOS)/$(GOARCH)/$(LIB_FILE): exports.go ../verify.go
	CGO_ENABLED=1 go build -o $@ -buildmode=c-shared $<
	mv lib/$(GOOS)/$(GOARCH)/$(LIB_PREFIX)$(LIB_NAME).h lib/$(GOOS)/$(GOARCH)/$(LIB_NAME).h || true	# Normalize header name

clean:
	rm -rf ../exports/lib/*
