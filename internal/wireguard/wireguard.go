// Package wireguard implements a few helpers for the WireGuard protocol
package wireguard

import (
	"errors"
	"fmt"
	"net"

	"github.com/eduvpn/eduvpn-common/internal/wireguard/ini"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func availableTCPPort() (int, error) {
	tcpaddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return -1, err
	}
	ltcp, err := net.ListenTCP("tcp", tcpaddr)
	if err != nil {
		return -1, err
	}
	defer ltcp.Close()
	return ltcp.Addr().(*net.TCPAddr).Port, nil
}

func availableUDPPort() (int, error) {
	udpaddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return -1, err
	}
	ludp, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		return -1, err
	}
	defer ludp.Close()
	return ludp.LocalAddr().(*net.UDPAddr).Port, nil
}

type Proxy struct {
	SourcePort int
	Listen     string
	Peer       string
}

func Config(cfg string, key *wgtypes.Key, tcp bool) (string, *Proxy, error) {
	// the key is nil if the client does not accept WireGuard
	if key == nil {
		return "", nil, errors.New("the server sent us a WireGuard profile but the client does not accept WireGuard")
	}

	var tcpp int
	var proxy string
	var err error

	if tcp {
		tcpp, err = availableTCPPort()
		if err != nil {
			return "", nil, err
		}
		udpp, err := availableUDPPort()
		if err != nil {
			return "", nil, err
		}
		proxy = fmt.Sprintf("127.0.0.1:%d", udpp)
	}

	rcfg, peer, err := configReplace(cfg, *key, proxy)
	if err != nil {
		return "", nil, err
	}
	var retP *Proxy
	if tcp {
		retP = &Proxy{
			SourcePort: tcpp,
			Listen:     proxy,
			Peer:       peer,
		}
	}
	return rcfg, retP, nil
}

// ConfigReplace replaces the wireguard config with our private key and proxy in case of TCP
func configReplace(cfg string, key wgtypes.Key, proxy string) (string, string, error) {
	// first parse the config
	secs := ini.Parse(cfg)
	if secs.Empty() {
		return "", "", errors.New("parsed ini is empty")
	}

	// find the interface section
	// and set the private key
	is, err := secs.Section("Interface")
	if err != nil {
		return "", "", err
	}
	is.AddOrReplaceKeyValue("PrivateKey", key.String())
	peer := ""
	if proxy != "" {
		ps, err := secs.Section("Peer")
		if err != nil {
			return "", "", err
		}
		peer, err = ps.RemoveKey("TCPEndpoint")
		if err != nil {
			return "", "", err
		}
		ps.AddOrReplaceKeyValue("Endpoint", proxy)
	}

	return secs.String(), peer, nil
}
