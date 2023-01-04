# Finite state machine

The eduvpn-common library uses a finite state machine internally to keep track of which state the client is in and to communicate data callbacks (e.g. to communicate the Authorization URL in the OAuth process to the client).

## Viewing the FSM
To view the FSM in an image, set the debug variable to `True`. This
outputs the graph with a `.graph` extension in the client-specified
config directory (See [API](../../api/index.html)). The format of this
graph is from [Mermaid](https://mermaid-js.github.io/mermaid/#/). You
can convert this to an image using the [Mermaid command-line client](https://github.com/mermaid-js/mermaid-cli) installed or from the Mermaid web site, the [Mermaid Live Editor](https://mermaid.live)

## FSM example
The following is an example of the FSM when the client has obtained a Wireguard/OpenVPN configuration from an eduVPN server

![](./fsm_example.svg)

The current state is highlighted in the <span style="color:cyan">cyan</span> color.

## State explanation
The states mean the following:
- `Deregistered`: the app is not registered with the wrapper
- `No_Server`: means the user has not chosen a server yet
- `Ask_Location`: the user selected a Secure Internet server but needs to choose a location
- `Search_Server`: the user is currently selecting a server in the UI
- `Loading_Server`: means we are loading the server details
- `Chosen_Server`: means the user has chosen a server to connect to
- `OAuth_Started`: means the OAuth process has started
- `Authorized`: means the OAuth process has finished and the user is now authorized with the server
- `Request_Config`: the user has requested a config for connecting
- `Ask_Profile`: the go code is asking for a profile selection from the UI
- `Disconnected`: the user has gotten a config for a server but is not connected yet
- `Disconnecting`: the OS is disconnecting and the Go code is doing the /disconnect
- `Connecting`: the OS is establishing a connection to the server
- `Connected`: the user has been connected to the server.
