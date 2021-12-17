# PHP Wrapper

## Requirements

You will need to install [PHP](https://www.php.net/downloads) 7.4 or later. For testing, you can use dependency
manager [Composer](https://getcomposer.org/doc/00-intro.md) to download PHPUnit.

Activate the [FFI](https://www.php.net/manual/en/ffi.setup.php) extension (Composer will also warn if you do not have it
enabled).

## Test etc.

Test (also installs PHPUnit using Composer and builds shared Go library for current platform):

```shell
make test
```

Only build shared library and copy modified C header for the current platform to the right directory:

```shell
make install-header
```

Or for the specified platform:

```shell
make install-header GOOS=windows GOARCH=amd64
```
