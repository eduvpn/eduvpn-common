# Code examples
This chapter contains code examples that use the API

## Go command line client
The following is an example [in the repository](https://codeberg.org/eduVPN/eduvpn-common/src/branch/main/cmd/cli/main.go). It is a command line client with the following flags
```
  -get-custom string
        The url of a custom server to connect to
  -get-institute string
        The url of an institute to connect to
  -get-secure string
        Gets secure internet servers
```
```go
{{#include ../../../cmd/cli/main.go}}
```
