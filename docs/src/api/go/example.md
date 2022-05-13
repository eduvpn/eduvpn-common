# Example with comments
The following is an example [in the repository](https://github.com/jwijenbergh/eduvpn-common/blob/main/cmd/cli/main.go). It is a command line client with the following flags
```
-get-institute string
  	The url of an institute to connect to
-get-secure string
  	Gets secure internet servers.
-get-secure-all string
  	Gets certificates for all secure internet servers. It stores them in ./certs. Provide an URL for the home server e.g. nl.eduvpn.org.
```
```go
{{#include ../../../../cmd/cli/main.go}}
```
