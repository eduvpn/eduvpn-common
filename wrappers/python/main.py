import eduvpncommon.main as eduvpn
import webbrowser


_eduvpn = eduvpn.EduVPN("org.eduvpn.app.linux", "configs")


@_eduvpn.event.on("SERVER_OAUTH_STARTED", eduvpn.StateType.Enter)
def oauth_initialized(url):
    print(f"Got OAUTH url {url}")
    webbrowser.open(url)


success = _eduvpn.register(debug=True)

if not success:
    print("failed to register")

print(_eduvpn.get_disco())

config, error = _eduvpn.connect("https://eduvpn.jwijenbergh.com")

if error:
    print("Got connect error", error)

print(config)
