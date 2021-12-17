.PHONY: build test clean

ifneq (clean,$(MAKECMDGOALS))
include ../../exports/platform.mk

ifeq (Windows_NT,$(OS))
SWIFT = ./swift.cmd
else
SWIFT = swift
endif
endif

build: .build_lib
	$(SWIFT) build --configuration release -Xlinker -L"../../exports/$(GOOS)/$(GOARCH)"

test: .build_lib
	$(SWIFT) test --parallel -Xlinker -L"../../exports/$(GOOS)/$(GOARCH)"

clean:
	rm -rf .build/ CEduVpnCommon/Sources/CEduVpnCommon/Headers/*.h

.build_lib:
	$(MAKE) -C ../../exports
	install "../../exports/$(GOOS)/$(GOARCH)/eduvpn_verify.h" -Dt CEduVpnCommon/Sources/CEduVpnCommon/Headers
