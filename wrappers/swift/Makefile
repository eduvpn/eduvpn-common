.PHONY: build test install-header clean

EXPORTS_PATH ?= ../../exports
include $(EXPORTS_PATH)/common.mk

ifeq ($(OS),Windows_NT)
SWIFT = ./swift.cmd
else
SWIFT = swift
endif

build: install-header
	$(SWIFT) build --configuration release -Xlinker -L"$(EXPORTS_LIB_SUBFOLDER_PATH)"

test: install-header
	$(SWIFT) test --parallel -Xlinker -L"$(EXPORTS_LIB_SUBFOLDER_PATH)"

install-header: .try-build-lib
	mkdir -p CEduVpnCommon/Sources/CEduVpnCommon/Headers
	cp "$(EXPORTS_LIB_SUBFOLDER_PATH)/$(LIB_NAME).h" CEduVpnCommon/Sources/CEduVpnCommon/Headers/  # Copy header for modulemap

clean:
	rm -rf .build/ CEduVpnCommon/Sources/CEduVpnCommon/Headers/*.h
