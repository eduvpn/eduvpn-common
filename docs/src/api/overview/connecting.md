# Connecting/Disconnecting
## Summary
Name: `Set Connected` and `Set Disconnected`

Arguments: None

Returns: `Error`

Used to signal to the FSM that we're connected/disconnected to the VPN

## Detailed information
This function is used to set the internal FSM state to connected. As the library does not actually connect to a VPN server, as this is platform specific, this must be called by the client to signal to the library that the user is connected to the VPN. If the FSM does not have a transition to the Connected state it will signal this with a returned error.

The same function is used to signal the FSM that the VPN is disconnected.
