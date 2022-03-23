import eduvpncommon.main as eduvpn
import webbrowser


_eduvpn = eduvpn.EduVPN("org.eduvpn.app.linux", "configs")


@_eduvpn.event.on("OAuthInitialized", eduvpn.StateType.Enter)
def oauth_initialized(url):
    print(f"Got OAUTH url {url}")
    webbrowser.open(url)


@_eduvpn.event.on("OAuthFinished", eduvpn.StateType.Enter)
def oauth_finished(data):
    print(f"Oauth finished {data}")


_eduvpn.register()
print(_eduvpn.get_disco())

config, error = _eduvpn.connect("https://eduvpn.jwijenbergh.com")
#
if error != "":
    print("Got connect error", error)

print(config)
