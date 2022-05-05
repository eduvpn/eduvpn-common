# Registering a client
## Summary
Name: `Register`

| Arguments | Description                             | type     |
| --------- | --------------------------------------  | -------- |
| Name      | Name of the client                      | string   |
| Directory | Path to save logging and state          | string   |
| Callback  | Function to be used as the FSM callback | function |
| Debug     | Indicates whether or not to configure debugging capabilities. See [this section](../../../gettingstarted/debugging/index.html) for more information on debugging. | boolean |

Returns: `Error`

Used as initialization function of the library
## Detailed information
This library is made to build eduVPN clients. To create such a client, the register method is used. This method takes a *name*, *directory* and *callback*. This method needs to be called whenever a client wants to use this library. If this method is not called then the remaining methods will not be available to use.

The *name* is the name of the client, also used as a client ID for OAuth. In general the name is the following for each official eduVPN client (documented [here](https://git.sr.ht/~fkooman/vpn-user-portal/tree/v3/item/src/OAuth/ClientDb.php)):


| Platform | Client ID                |
| -------- | ------------------------ |
| Linux    | `org.eduvpn.app.linux`   |
| Windows  | `org.eduvpn.app.windows` |
| MacOS    | `org.eduvpn.app.macos`   |
| Android  | `org.eduvpn.app.android` |
| iOS      | `org.eduvpn.app.ios`     |

The *directory* is the file path where logging and config files are stored. The library creates this directory if it doesn't exist. This can be an absolute or relative path. We recommend to use an absolute path to ensure that the right directory is chosen.

The *callback* is the function that gets called when the internal Finite State Machine switches state. This callback function must consist of three arguments

- Old state: The old state as a string, which is the current FSM state before the transition. See [FSM states](../../gettingstarted/debugging/fsm.html#state-explanation) for a list of states.
- New state: The current state for the FSM after the transition, also a string. See [FSM states](../../gettingstarted/debugging/fsm.html#state-explanation) for a list of states.
- Data: The data that gets sent by the library as a string. Most common this is JSON data to build the UI or in case of OAuth it is the authorization URL that needs to be opened by the browser. When there is no data this is an empty string.
