.PHONY: build test test-go test-wrappers clean

build:
	$(MAKE) -C exports build

test: test-go test-wrappers

test-go:
	go test

wrappers = $(wildcard wrappers/*/)

# Enable parallelism if -j is specified
test-wrappers: build
	$(MAKE) $(foreach wrapper,$(wrappers),.test_$(wrapper))

clean:
	$(MAKE) .clean_libs $(foreach wrapper,$(wrappers),.clean_$(wrapper))

.clean_libs:
	$(MAKE) -C exports clean

define wrapper_targets
.test_$(1):
	$(MAKE) -C $(1) test
.clean_$(1):
	$(MAKE) -C $(1) clean
endef

$(foreach wrapper,$(wrappers),$(eval $(call wrapper_targets,$(wrapper))))
