# Example with Comments

```python
import eduvpncommon.main as eduvpn

# Callbacks
@_eduvpn.event.on("OAuth_Started", eduvpn.StateType.Enter)
def oauth_initialized(url):
	# Open the webbrowser with the url
    webbrowser.open(url)


@_eduvpn.event.on("Ask_Profile", eduvpn.StateType.Enter)
def ask_profile(profiles):
	# Set a profile
    _eduvpn.set_profile("example")
	
# Register the state
# We use linux so the client ID will be org.eduvpn.app.linux
# We want to store the config files in configs
# And enable debugging
_eduvpn = eduvpn.EduVPN("org.eduvpn.app.linux", "configs")
register_err = _eduvpn.register(debug=True)

if register_err:
	# Handle error

# Connect to eduvpn.example.com
config, config_type, config_err = _eduvpn.get_connect_config("eduvpn.example.com", False)

if config_err:
	# Handle error
	
if config_type == "wireguard":
	# Connect using wireguard with the config
elif config_type == "openvpn":
    # Connect using OpenVPN with the config
else:
	# Handle error

# Set connected
set_connect_err = _eduvpn.set_connected()
if set_connect_err:
	# Handle error

# Handle cleanup
_eduvpn.deregister()
```
