.PHONY: build pack test clean

EXPORTS_PATH ?= ../../exports
EXPORTS_LIB_PATH ?= $(EXPORTS_PATH)/lib

build:
	mvn --no-transfer-progress compile -DEXPORTS_LIB_PATH="$(EXPORTS_LIB_PATH)"

pack:
	mvn --no-transfer-progress package -DEXPORTS_LIB_PATH="$(EXPORTS_LIB_PATH)"

test:
ifneq ($(EXPORTS_PATH),)
ifneq ($(wildcard $(EXPORTS_PATH)/Makefile),)
	$(MAKE) -C "$(EXPORTS_PATH)"
endif
endif
	mvn --no-transfer-progress test -DEXPORTS_LIB_PATH="$(EXPORTS_LIB_PATH)"

clean:
	rm -rf target/
