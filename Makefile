.PHONY: build test test-go test-wrappers clean

build:
	$(MAKE) -C exports build

test: test-go test-wrappers

test-go:
	go test

test-wrappers: build
	$(MAKE) -C wrappers/csharp test

clean:
	$(MAKE) -C exports clean
	$(MAKE) -C wrappers/csharp clean
