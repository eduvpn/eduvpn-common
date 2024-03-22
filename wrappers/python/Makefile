.DEFAULT_GOAL := pack
.PHONY: install-lib pack test clean lint fmt

VERSION := $(shell grep -o 'const Version = "[^"]*' ../../internal/version/version.go | cut -d '"' -f 2)

install-lib:
	rm -rf eduvpn_common/lib/*
	install "../../lib/libeduvpn_common-${VERSION}.so" -Dt "eduvpn_common/lib"

# Build for current platform only
pack: install-lib
	python3 -m build --sdist --wheel .

test: install-lib
	python3 -m unittest tests

clean:
	rm -rf build/ dist/ *.egg-info/ eduvpn_common/lib/* venv

lint:
	ruff check eduvpn_common
	ruff format --check eduvpn_common

fmt:
	ruff format eduvpn_common
