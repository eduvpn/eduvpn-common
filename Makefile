.PHONY: build test test-go test-wrappers

build:
	$(MAKE) -C exports build

test: test-go test-wrappers

test-go:
	go test

test-wrappers: build
	$(MAKE) -C wrappers/csharp test
