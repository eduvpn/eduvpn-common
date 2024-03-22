.DEFAULT_GOAL := build
.PHONY: build fmt cli clean


VERSION := $(shell grep -o 'const Version = "[^"]*' internal/version/version.go | cut -d '"' -f 2)


build:
	CGO_ENABLED="1" go build -o lib/libeduvpn_common-${VERSION}.so -buildmode=c-shared ./exports

fmt:
	gofumpt -w .
	ruff format wrappers/python/eduvpn_common

lint:
	golangci-lint run -E stylecheck,revive,gocritic ./...
	ruff check wrappers/python/eduvpn_common
	ruff format --check wrappers/python/eduvpn_common

cli:
	go build -o eduvpn-common-cli ./cmd/cli

test:
	go test ./...

clean:
	rm -rf lib
	go clean
