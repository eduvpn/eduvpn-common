.PHONY: build test test-go test-wrappers clean

build:
	$(MAKE) -C exports

test: test-go test-wrappers

test-go:
	go test ./...

#WRAPPERS ?= $(notdir $(patsubst %/,%,$(wildcard wrappers/*/)))
WRAPPERS=python

# Enable parallelism if -j is specified, but first execute build
test-wrappers: build
	$(MAKE) $(foreach wrapper,$(WRAPPERS),.test-$(wrapper))

clean: .clean-libs $(foreach wrapper,$(WRAPPERS),.clean-$(wrapper))

.clean-libs:
	$(MAKE) -C exports clean

# Define test & clean for each wrapper
define wrapper_targets
.test-$(1):
	$(MAKE) -C wrappers/$(1) test
.clean-$(1):
	$(MAKE) -C wrappers/$(1) clean
endef
$(foreach wrapper,$(WRAPPERS),$(eval $(call wrapper_targets,$(wrapper))))
