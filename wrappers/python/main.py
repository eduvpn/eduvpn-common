import eduvpncommon.main as eduvpn


_eduvpn = eduvpn.EduVPN("org.eduvpn.app.linux", "configs")


@_eduvpn.event.on("REGISTERED", eduvpn.StateType.Enter)
def registered(data):
    print(f"REGISTERED PYTHON WITH DATA {data}")


_eduvpn.register()
