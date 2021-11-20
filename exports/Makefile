.PHONY: build

ifeq ($(OS),Windows_NT)
    lib_suffix := .dll
else
    lib_suffix := .so
endif

build: eduvpn_verify$(lib_suffix)

eduvpn_verify.dll eduvpn_verify.so: exports.go ../verify.go
	go build -o $@ -buildmode=c-shared $<
