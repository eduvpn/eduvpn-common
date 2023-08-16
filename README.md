# eduVPN shared library

This repository contains a Go library with functions that all eduVPN clients can use. The goal is to let eduVPN clients
link against this library and gradually merge more common logic between eduVPN clients into this repository.

[Cgo](https://pkg.go.dev/cmd/cgo) is used to build the Go library into a shared dynamic library. Wrappers were
written using some FFI framework for each language used in eduVPN clients to easily interface with the library.

The only support language is Python at the moment. Other languages will come with updates.

## Documentation
The documentation for this library can be found at [GitHub pages](https://eduvpn.github.io/eduvpn-common).

## Contributing
Contributions are welcome. Helping with translations can be done on weblate:
<a href="https://hosted.weblate.org/engage/eduvpn-common/">
<img src="https://hosted.weblate.org/widget/eduvpn-common/eduvpn-common/multi-auto.svg" alt="Translation status" />
</a>

## License
[MIT](./LICENSE)

## Authors
This work is done by [@stevenwdv](https://github.com/stevenwdv) and [@jwijenbergh](https://github.com/jwijenbergh) at the [SURF](https://www.surf.nl/) and [GÉANT](https://geant.org/) organization.
