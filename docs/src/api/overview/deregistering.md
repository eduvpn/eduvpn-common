# Deregistering
## Summary
name: `Deregister`

Arguments: None

Returns: Nothing

Used to cleanup the library by deregistering the client and saving the config files
## Detailed information
The deregister method is used to cleanup the library. It should be called when the client closes. This also saves the state to the directory that was passed in the [Register](./registering.html) method.
