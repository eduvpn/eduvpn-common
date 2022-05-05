# Go
The API that has no additional wrapper code is the Go API. To begin to use the Go library in a Go client you first need to import it:

```go
import "github.com/jwijenbergh/eduvpn-common"
```

This brings the library into scope using the eduvpn-common prefix.

The functions that we define all operate on a `VPNState` object, thus to call a function it needs to be first created and then the function needs to be called. An example of how to tie all of this together is done at the end.
