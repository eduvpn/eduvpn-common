# API overview

This section defines the API in high-level, we explain what functions there are, what their use is and what a typical flow is for creating an eduVPN client with this library. The language specific documentation will be given in separate sections.

## Note on types and names
This section acts as an introduction to the API, as such this section will e.g. only give general typing information for the arguments and return values. Please read the language specific API documentation as well. To give an example, we will often say that an `Error` is returned. For Go this is the `error` type, whereas for Python this is simply a string with the error message.

Additionally, the name of the function described will not be stated exactly as this has language specific differences. For example in Go we use the camel case construct, whereas for python snake case is used. E.g. compare `GetConnectConfig` (Go) and `get_connect_config` (Python)
