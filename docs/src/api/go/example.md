# Example with Comments

```go

// Bring the library into scope with the eduvpn prefix
import eduvpn "github.com/jwijenbergh/eduvpn-common"

// Callbacks

func stateCallback(state *eduvpn.VPNState, oldState string, newState string, data string) {

	// OAuth is started, open the browser with the authorization URL
	if newState == "OAuth_Started" {
		openBrowser(data)
	}
	
	// Multiple profiles are found, we need to send a profile ID back using state.SetProfileID
	if newState == "Ask_Profile" {
		selectAndSendProfile(state, data)
	}
}

func main() {
	// Create the VPNState
	state := &eduvpn.VPNState{}
	
	// Register the state
	// We use linux so the client ID will be org.eduvpn.app.linux
	// We want to store the config files in configs
	// We wrap the callback with the state argument
	// And enable debugging
	registerErr := state.Register("org.eduvpn.app.linux", "configs", func(old string, new string, data string) {
		stateCallback(state, old, new, data)
	}, true)
	
	if registErr != nil {
		// handle the error of not being able to register
	}
	
	// Cleanup the library at the end
	defer state.Deregister()
	
	// Connect to an example server without forcing TCP
	config, configType, configErr := state.GetConnectConfig("eduvpn.example.com", false)
	
	if configErr != nil {
		// handle the error of not being able to get a config
	}
	
	if configType == "wireguard" {
		// Connect using wireguard with the config
	} else {
	    // Connect using OpenVPN with the config
	}
	
	// We are connected
	setConnectErr := state.SetConnected()
	
	if setConnectErr != nil {
		// handle the error of not being able to call set connected
	}
}
```
