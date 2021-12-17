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

exports_dir = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
ifeq (Windows_NT,$(OS))
export PATH := $(exports_dir)/$(GOOS)/$(GOARCH):$(PATH)
else
export LD_LIBRARY_PATH := $(exports_dir)/$(GOOS)/$(GOARCH):$(LD_LIBRARY_PATH)
endif
