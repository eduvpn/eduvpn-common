.PHONY: pack test clean

EXPORTS_PATH ?= ../../exports
EXPORTS_LIB_PATH ?= $(EXPORTS_PATH)/lib

ifneq ($(MAKECMDGOALS),clean)
include $(EXPORTS_PATH)/platform.mk
endif

ifdef PLAT_NAME
SETUP_ARGS += --plat-name=$(PLAT_NAME)
endif

# Build for current platform only
pack:
	./setup.py bdist_wheel $(SETUP_ARGS) --exports-lib-path="$(EXPORTS_LIB_PATH)"

test:
ifneq ($(EXPORTS_PATH),)
ifneq ($(wildcard $(EXPORTS_PATH)/Makefile),)
	$(MAKE) -C "$(EXPORTS_PATH)"
endif
endif
	install "$(EXPORTS_LIB_PATH)/$(GOOS)/$(GOARCH)/$(LIB_FILE)" -Dt "eduvpncommon/lib"
	python3 -m unittest test_discovery
	rm eduvpncommon/lib/*

clean:
	rm -rf build/ dist/ *.egg-info/ lib/*
