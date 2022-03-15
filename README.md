# EduVPN shared library

This repository contains a Go library with functions that all eduVPN clients can use. The goal is to let eduVPN clients
link against this library and gradually merge more common logic between eduVPN clients into this repository.

[cgo](https://pkg.go.dev/cmd/cgo) is used to build the Go library into a shared dynamic library. Wrappers were
written using some FFI framework for each language used in eduVPN clients to easily interface with the library.

Supported languages:
- Android (Java)
- C#
- Php
- Python
- Swift

## Documentation
The documentation for this library can be found at x.

## Contributing
Contributions are welcome.

## License
This code does not have a license yet.
