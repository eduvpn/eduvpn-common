package client

import (
	"encoding/json"

	"codeberg.org/eduVPN/proxyguard"

	"github.com/eduvpn/eduvpn-common/i18nerr"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/types/cookie"
)

// ProxyLogger is defined here such that we can update the proxyguard logger
type ProxyLogger struct{}

// Logf logs a message with parameters
func (pl *ProxyLogger) Logf(msg string, params ...interface{}) {
	log.Logger.Infof("[Proxyguard] "+msg, params...)
}

// Log logs a message
func (pl *ProxyLogger) Log(msg string) {
	log.Logger.Infof("[Proxyguard] %s", msg)
}

// StartProxyguard starts proxyguard for proxied WireGuard connections
func (c *Client) StartProxyguard(ck *cookie.Cookie, listen string, tcpsp int, peer string, gotFD func(fd int, pips string), ready func()) error {
	var err error
	proxyguard.UpdateLogger(&ProxyLogger{})

	proxyc := proxyguard.Client{
		Listen:        listen,
		TCPSourcePort: tcpsp,
		SetupSocket: func(fd int, pips []string) {
			if gotFD == nil {
				return
			}
			b, err := json.Marshal(pips)
			if err != nil {
				log.Logger.Errorf("marshalling peer IPs failed: %v", err)
				return
			}
			gotFD(fd, string(b))
		},
		Ready: ready,
	}

	// we set peer IPs to nil here as proxyguard already does a DNS request for us
	err = proxyc.Tunnel(ck.Context(), peer, nil)
	if err != nil {
		return i18nerr.Wrap(err, "The VPN proxy exited")
	}
	return err
}
