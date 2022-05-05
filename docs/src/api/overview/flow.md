# Typical flow
A typical flow of creating a client is calling the methods that we talked about in order that we introduced them:

- The client starts, it registers with the library
- A list of discovery servers/organizations is obtained using the library
- The client selects an URL to connect to and calls the function to get an OpenVPN/Wireguard config from the library
- The client uses the OS specific libraries and programs to use the OpenVPN/Wireguard config to establish a tunnel and calls the function to connect or disconnect
- When the client is done it calls the deregister method to save all configuration and clean up
