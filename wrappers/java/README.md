# Java wrapper

## Requirements

You will need to install JDK 8 or later ([Adoptium](https://adoptium.net/)
or [Oracle](https://www.oracle.com/java/technologies/downloads/)). To easily compile the project, you should
download [Maven](https://maven.apache.org/).

## Build & test

First build the shared Go library. Next:

Build `EduVpnCommon`:

```shell
make
```

Build as JAR, including shared Go library:

```shell
make pack
```

The JAR will include all versions of the library that are built in the `exports` folder.

If you do not build this as part of the full repository, specify `EXPORTS_PATH="path/to/exports-folder"`
when calling make. This folder must contain `common.mk` and the `lib/` folder with built libraries.

Test:

```shell
make test
```
