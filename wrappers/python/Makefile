.PHONY: pack test clean

EXPORTS_PATH ?= ../../exports
include $(EXPORTS_PATH)/common.mk

ifdef PLAT_NAME
SETUP_ARGS += --plat-name=$(PLAT_NAME)
endif

# Build for current platform only
pack:
	./setup.py bdist_wheel $(SETUP_ARGS) --exports-lib-path="$(EXPORTS_LIB_PATH)"

test: .try_build_lib
	install "$(EXPORTS_LIB_SUBFOLDER_PATH)/$(LIB_FILE)" -Dt "eduvpncommon/lib"
	python3 -m unittest test_discovery
	rm eduvpncommon/lib/*

clean:
	rm -rf build/ dist/ *.egg-info/ lib/*
