# Go <-> language X interop
Because this library is meant to be a *general* library for other clients to use that are written in different programming languages, we need to find a way to make this Go library available on each platform and codebase. The approach that we take is to build a C library from the Go library using Cgo. Cgo can have its disadvantages with performance and the constant conversion between Go and C types. To overcome those barriers, this library has the following goals (with some others noted here):
- **Be high-level**. Functions should do as much as possible in Go. The exported API should fit in one file. Lots of low-level functions would be a constant conversion between C and Go which adds overhead
- **Move as much state to Go as possible**. For example, Go keeps track of the servers you have configured and discovery. This makes the arguments to functions simple, clients should pass simple identifiers that Go can look up in the state
- **Easy type conversion**: to convert between C and Go types, JSON is used. Whereas Protobuf, Cap'n'proto or flatbuffers are more performant, they are harder to debug, add thousands of lines of autogenerated code and are not human friendly. Using JSON, the clients can approach it the same way they would use with a server using a REST API. Another approach is to just  convert from Go -> C types -> language types. This was tried in version 1 of the library, but this ended up being too much work and manual memory management
- **Make it as easy as possible for clients to manage UI and internal state**: we use a state machine that gives the clients information in which state the Go library is in, e.g. we're selecting a server profile, we're loading the server endpoints. This library is not only a layer to talk to eduVPN servers, but the whole engine for a client
- **Implement features currently not present in existing clients**: WireGuard to OpenVPN failover, WireGuard over TCP
- **Follow the official eduVPN specification** and also contribute changes when needed
- **Secure**: We aim to follow the latest OAuth recommendations, to not store secret data and e.g. disable OpenVPN scripts from being ran by default

And finally the most important goal:
- **The advantages that this library brings for clients should outweigh the cost of incorporating it into the codebase**. Initial versions would take more work than we get out of it. However, when each eduVPN/Let's Connect! client uses this library we should expect a net gain. New features should be easier to implement for clients by simply requiring a new eduvpn-common version and using the necessary functions