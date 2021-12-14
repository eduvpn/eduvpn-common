.PHONY: build test clean

ifneq (clean,$(MAKECMDGOALS))
GOOS      != go env GOHOSTOS
GOARCH    != go env GOHOSTARCH

ifeq (Windows_NT,$(OS))
export PATH := $(PATH):$(abspath ../../exports/$(GOOS)/$(GOARCH))
else
export LD_LIBRARY_PATH := $(LD_LIBRARY_PATH):$(abspath ../../exports/$(GOOS)/$(GOARCH))
endif
endif

build:
	$(MAKE) -C ../../exports
ifeq (Windows_NT,$(OS))
	"$(COMSPEC)" /c "\".\\fix-path.cmd & swift build --configuration release -Xlinker -L^\"../../exports/$(GOOS)/$(GOARCH)^\"\""
else
	swift build --configuration release -Xlinker -L"../../exports/$(GOOS)/$(GOARCH)"
endif

test:
	$(MAKE) -C ../../exports
ifeq (Windows_NT,$(OS))
	"$(COMSPEC)" /c "\".\\fix-path.cmd & swift test --parallel -Xlinker -L^\"../../exports/$(GOOS)/$(GOARCH)^\"\""
else
	swift test --parallel -Xlinker -L"../../exports/$(GOOS)/$(GOARCH)"
endif

clean:
	rm -rf .build/
