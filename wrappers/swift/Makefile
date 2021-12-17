.PHONY: build test clean

ifneq (clean,$(MAKECMDGOALS))
GOOS      != go env GOHOSTOS
GOARCH    != go env GOHOSTARCH

ifeq (Windows_NT,$(OS))
SWIFT = ./swift.cmd
export PATH := $(abspath ../../exports/$(GOOS)/$(GOARCH)):$(PATH)
else
SWIFT = swift
export LD_LIBRARY_PATH := $(abspath ../../exports/$(GOOS)/$(GOARCH)):$(LD_LIBRARY_PATH)
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
