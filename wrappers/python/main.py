import eduvpncommon.main as eduvpn

eduvpn.Register("org.eduvpn.app.linux", "configs", eduvpn.state_change)

print(eduvpn.GetDiscoServers())
