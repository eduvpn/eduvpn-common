# Android (Java) wrapper

## Requirements

You will need to install the JDK compatible with Gradle and the Android Gradle plugin. Gradle specifies
a [maximum supported JDK version](https://docs.gradle.org/current/userguide/compatibility.html), while the Android
Gradle plugin specifies a [minimum supported JDK version](https://developer.android.com/studio/releases/gradle-plugin) (
see 'Compatibility' table for the right Gradle version). Additionally, the Android Gradle plugin requires
a [certain Gradle version range](https://developer.android.com/studio/releases/gradle-plugin#updating-gradle). Lastly, (
older versions of) Android Studio and especially IntelliJ Ultimate may not support some newer Android Gradle plugin
versions.

If you see `Unsupported class file major version xx` then Gradle wants you to use an older Java version. If you
see `Android Gradle plugin requires Java xx to run.` then the Android Gradle plugin wants you to use a newer Java
version. Set `JAVA_HOME` to the right JDK install.

See the [list of Gradle releases](https://github.com/gradle/gradle/releases)
and [list of Android Gradle plugin releases](https://maven.google.com/web/?q=com.android.tools.build#com.android.tools.build:gradle)
.

Versions are managed per project by
the [Gradle wrapper](https://docs.gradle.org/current/userguide/gradle_wrapper.html) (see `gradle*` files).
Run `./gradlew --version` to get the project Gradle version. Look at the dependencies in `/build.gradle` for the current
Android Gradle plugin version. This means that you do not need to install Gradle separately yourself.

Currently, versions are Gradle 7.0.2 and Android Gradle plugin 7.0.4, which means you have to install a **JDK with a
version between 11 and 16**. If desired, these can be upgraded to e.g. 7.4 and 7.1.1 respectively without problems, but
as mentioned, IDE support may be limited with newer versions. The Gradle wrapper may be updated using
e.g. `./gradlew wrapper --gradle-version 7.4`. After that, the Android Gradle plugin may be updated by changing the
version in `/build.gradle`.

You will also need the Android SDK, which comes with [Android Studio](https://developer.android.com/studio/).

## Build & test

Build AAR (Gradle will also run unit tests):

```shell
make
```

This will build an AAR in `lib/build/outputs/aar`, which will include the shared Go library for all Android
architectures, which it will build using the Android NDK via CMake, which calls `make` with the right compiler.

Run unit tests without an Android emulator:

```shell
make unit-test
```

This will build the library for your current desktop OS and use that.

Run Android instrumented tests on a new emulator:

```shell
make android-test
```

This uses [Gradle managed virtual devices](https://developer.android.com/studio/preview/features#gmd). This experimental
feature is enabled in `gradle.properties`. It is normal that tests that pass are not logged.

Run Android instrumented tests on an already running emulator:

```shell
make connected-android-test
```

This will be faster when used multiple times, as the emulator is reused.

Run both unit tests and Android instrumented tests on a new emulator:

```shell
make test
```

For all commands you can specify options to pass to Gradle via `GRADLE_FLAGS=`, e.g. `GRADLE_FLAGS=--info`.
Specify `NO_DAEMON=1` to add `--no-daemon`.

## Notes

The same Java code is used for the Android instrumented tests as for the unit tests. Both use Java resources that are
copied from the `../../test_data` folder by Gradle.

This library uses JNA, not JNI. Hence, there is no C wrapper. The library is dynamically opened with `dlopen`
via `libjnidispatch.so` which comes with the JNA AAR.

If you want to know how the tests would look like with JUnit 5, or if for some reason you want to look at a pure Java
wrapper using Maven,
see [`b60ecf2`](https://github.com/stevenwdv/eduvpn-common/tree/b60ecf2fe5ddfe506e02093286b3931873187e91/wrappers/java).
