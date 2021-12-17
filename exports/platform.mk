ifndef GOOS
export GOOS   != go env GOHOSTOS
endif
ifndef GOARCH
export GOARCH != go env GOHOSTARCH
endif

ifeq (windows,$(GOOS))
LIB_PREFIX =
LIB_SUFFIX = .dll
else ifeq (darwin,$(GOOS))
LIB_PREFIX = lib
LIB_SUFFIX = .dylib
else
LIB_PREFIX = lib
LIB_SUFFIX = .so
endif

ifeq (Windows_NT,$(OS))
export PATH := $(abspath ../../exports/$(GOOS)/$(GOARCH)):$(PATH)
else
export LD_LIBRARY_PATH := $(abspath ../../exports/$(GOOS)/$(GOARCH)):$(LD_LIBRARY_PATH)
endif
