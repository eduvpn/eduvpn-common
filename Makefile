.DEFAULT_GOAL := build
.PHONY: build fmt lint cli test clean coverage sloc


VERSION := $(shell grep -o 'const Version = "[^"]*' internal/version/version.go | cut -d '"' -f 2)


build:
	CGO_ENABLED="1" go build -o lib/libeduvpn_common-${VERSION}.so -buildmode=c-shared ./exports

fmt:
	gofumpt -w .

lint:
	golangci-lint run -E stylecheck,revive,gocritic ./...

cli:
	go build -o eduvpn-common-cli ./cmd/cli

test:
	go test -tags=cgotesting -race ./...

clean:
	rm -rf lib
	go clean

coverage:
	go test -tags=cgotesting -v -coverpkg=./... -coverprofile=common.cov ./...
	go tool cover -func common.cov

sloc:
	tokei --exclude "*_test.go" -t=Go . || cloc --include-ext=go --not-match-f='_test.go' .
