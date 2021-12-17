.PHONY: build copy-to clean

include platform.mk

# Creates targets like 'linux/amd64/eduvpn_verify.so'
build: $(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify$(LIB_SUFFIX)

$(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify$(LIB_SUFFIX): exports.go ../verify.go
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ -buildmode=c-shared $<
	mv $(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify.h $(GOOS)/$(GOARCH)/eduvpn_verify.h || true

copy-to: $(GOOS)/$(GOARCH)/$(LIB_PREFIX)eduvpn_verify$(LIB_SUFFIX)
	install $< -Dt "$(COPY_TARGET)"

clean:
	rm -rf ../exports/*/
