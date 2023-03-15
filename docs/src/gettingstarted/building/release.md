# Building for release

To build for release, make sure you obtain the tarball artifacts in the release (ending with `.tar.xz`) at <https://github.com/eduvpn/eduvpn-common/releases>.

These are signed with minisign and gpg keys, make sure to verify these signatures using the public keys available here: <https://github.com/eduvpn/eduvpn-common/tree/main/keys>, they are also available externally:
- <https://app.eduvpn.org/linux/v4/deb/app+linux@eduvpn.org.asc>
- <https://git.sr.ht/~jwijenbergh/python3-eduvpn-common.rpm/tree/main/item/SOURCES/minisign-CA9409316AC93C07.pub>

To build for release, make sure to extract the tarball, and then add `-tags=release` to the `GOFLAGS` environment variable:

```bash
GOFLAGS="-tags=release" make
```

Proceed the build like normally.
