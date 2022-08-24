package server

func (servers *Servers) RemoveCustomServer(url string) {
	servers.CustomServers.Remove(url)
}

