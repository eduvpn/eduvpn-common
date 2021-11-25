# EduVPN shared library

This repository contains a Go library with functions that all EduVPN clients can use. The goal is to let EduVPN clients
link against this library and gradually merge more common logic between EduVPN clients into this repository.

cgo is used to build the go library into a shared dynamic library. Wrappers will be written using some FFI framework for
each language used in EduVPN clients to easily interface with the library.

## Functionality

Currently, only verification of signatures on files from `disco.eduvpn.org` is supported. For now, these files have to
be downloaded by the caller.

## Build & test

Build shared library for current platform:
```shell
make
```

Build shared library for specified OS & architecture (example):
```shell
make OS=windows ARCH=386
```

Results will be output in `exports/`.

Test Go code:
```shell
make test-go
```

Test wrappers:
```shell
make test-wrappers
```

Take a look at `wrappers/<lang>/` for descriptions per wrapper.

## Directory

- `verify.go`: main API
- `verify_test.go` and `test_data/`: tests for API
- `exports/`: C API interface
- `wrappers/`: Wrappers per language, more will be added
