.PHONY: build test clean

ifdef PLAT_NAME
SETUP_ARGS += --plat-name=$(PLAT_NAME)
endif

# Build for current platform only
build:
	./setup.py bdist_wheel $(SETUP_ARGS)

test:
	$(MAKE) -C ../../exports copy-to COPY_TARGET=../wrappers/python/eduvpncommon/lib
	python3 -m unittest test_discovery
	rm eduvpncommon/lib/*

clean:
	rm -rf build/ dist/ *.egg-info/ lib/*
