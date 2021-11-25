First build the shared Go library. Next:

Build `EduVpnCommon`:
```shell
make
```

Build as nupkg, including eduvpn_verify library:
```shell
make pack
```

Currently, directly referencing the project may not work if you have multiple dynamic libraries compiled in
the `exports` folder. If you instead add the `.nupkg`, e.g. with one of the
methods [here](https://stackoverflow.com/q/43400069) or [here](https://stackoverflow.com/q/10240029), it automatically
copies the correct library.

Test:
```shell
make test
```
