.PHONY: build test install-header clean

EXPORTS_PATH ?= ../../exports
EXPORTS_LIB_PATH ?= $(EXPORTS_PATH)/lib

ifneq ($(MAKECMDGOALS),clean)
include $(EXPORTS_PATH)/platform.mk

LIB_DIR = $(EXPORTS_LIB_PATH)/$(GOOS)/$(GOARCH)

ifeq ($(OS),Windows_NT)
SWIFT = ./swift.cmd
else
SWIFT = swift
endif
endif

build: install-header
	$(SWIFT) build --configuration release -Xlinker -L"$(LIB_DIR)"

test: install-header
	$(SWIFT) test --parallel -Xlinker -L"$(LIB_DIR)"

install-header:
ifneq ($(EXPORTS_PATH),)
ifneq ($(wildcard $(EXPORTS_PATH)/Makefile),)
	$(MAKE) -C "$(EXPORTS_PATH)"
endif
endif
	install "$(LIB_DIR)/$(LIB_NAME).h" -Dt CEduVpnCommon/Sources/CEduVpnCommon/Headers  # Copy header for modulemap

clean:
	rm -rf .build/ CEduVpnCommon/Sources/CEduVpnCommon/Headers/*.h
