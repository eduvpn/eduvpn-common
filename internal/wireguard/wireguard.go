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

// Proxy is the proxyguard information
type Proxy struct {
	// SourcePort is the source port of the TCP socket
	SourcePort int
	// Listen is the IP:PORT of the udp listener
	Listen string
	// Peer is the hostname/ip:port of the WireGuard peer
	Peer string
}

// Config gets a wireguard config with API config `cfg`, wg key `key` and whether to use proxyguard `proxy`
func Config(cfg string, key *wgtypes.Key, proxy bool) (string, *Proxy, error) {
	// the key is nil if the client does not accept WireGuard
	if key == nil {
		return "", nil, errors.New("the server sent us a WireGuard profile but the client does not accept WireGuard")
	}

	var tcpp int
	var plisten string
	var err error

	if proxy {
		tcpp, err = availableTCPPort()
		if err != nil {
			return "", nil, err
		}
		udpp, err := availableUDPPort()
		if err != nil {
			return "", nil, err
		}
		plisten = fmt.Sprintf("127.0.0.1:%d", udpp)
	}

	rcfg, peer, err := configReplace(cfg, *key, plisten)
	if err != nil {
		return "", nil, err
	}
	var retP *Proxy
	if proxy {
		retP = &Proxy{
			SourcePort: tcpp,
			Listen:     plisten,
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
		peer, err = ps.RemoveKey("ProxyEndpoint")
		if err != nil {
			return "", "", err
		}
		ps.AddOrReplaceKeyValue("Endpoint", proxy)
	}

	return secs.String(), peer, nil
}
