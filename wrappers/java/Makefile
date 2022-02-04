.PHONY: build pack test clean

EXPORTS_PATH ?= ../../exports
include $(EXPORTS_PATH)/common.mk

build:
	mvn --no-transfer-progress compile -DEXPORTS_LIB_PATH="$(EXPORTS_LIB_PATH)"

pack:
	mvn --no-transfer-progress package -DEXPORTS_LIB_PATH="$(EXPORTS_LIB_PATH)"

test: .try_build_lib
	mvn --no-transfer-progress test -DEXPORTS_LIB_PATH="$(EXPORTS_LIB_PATH)"

clean:
	rm -rf target/
