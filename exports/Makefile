.PHONY: build copy-to clean

GOOS      != go env GOHOSTOS
GOARCH    != go env GOHOSTARCH

ifeq (windows,$(GOOS))
LIB_PREFIX =
LIB_SUFFIX = .dll
else ifeq (darwin,$(GOOS))
LIB_PREFIX = lib
LIB_SUFFIX = .dylib
else
LIB_PREFIX = lib
LIB_SUFFIX = .so
endif

# Creates targets like 'linux/amd64/eduvpn_verify.so'
build: $(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify$(LIB_SUFFIX)

$(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify$(LIB_SUFFIX): exports.go ../verify.go
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ -buildmode=c-shared $<
	cp $(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify.h ./eduvpn_verify.h

copy-to: $(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify$(LIB_SUFFIX)
	install $< -Dt "$(COPY_TARGET)"

clean:
	rm -rf ../exports/*/
