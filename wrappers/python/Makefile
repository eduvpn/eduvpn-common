.PHONY: pack test clean

EXPORTS_PATH ?= ../../exports
include $(EXPORTS_PATH)/common.mk

ifdef PLAT_NAME
override SETUP_ARGS += --plat-name=$(PLAT_NAME)
endif

# Build for current platform only
pack:
	mkdir -p eduvpncommon/lib
	./setup.py bdist_wheel $(SETUP_ARGS) --exports-lib-path="$(EXPORTS_LIB_PATH)"

test: .try-build-lib
	install "$(EXPORTS_LIB_SUBFOLDER_PATH)/$(LIB_FILE)" -Dt "eduvpncommon/lib"
	python3 -m unittest test_discovery
	rm eduvpncommon/lib/*

clean:
	rm -rf build/ dist/ *.egg-info/ lib/*
ifeq ($(CLEAN_ALL),1)
	rm -rf venv/
endif
