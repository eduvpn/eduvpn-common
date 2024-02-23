package client

import (
	"codeberg.org/eduVPN/proxyguard"
	"github.com/eduvpn/eduvpn-common/i18nerr"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/types/cookie"
)

// ProxyLogger is defined here such that we can update the proxyguard logger
type ProxyLogger struct{}

// Logf logs a message with parameters
func (pl *ProxyLogger) Logf(msg string, params ...interface{}) {
	log.Logger.Debugf(msg, params...)
}

// Log logs a message
func (pl *ProxyLogger) Log(msg string) {
	log.Logger.Debugf("%s", msg)
}

// StartProxyguard starts proxyguard for proxied WireGuard connections
func (c *Client) StartProxyguard(ck *cookie.Cookie, listen string, tcpsp int, peer string, gotFD func(fd int), ready func()) error {
	var err error
	proxyguard.UpdateLogger(&ProxyLogger{})
	proxyguard.GotClientFD = gotFD
	proxyguard.ClientProxyReady = func() {
		// already connected
		// no need to signal to the client that the proxy is ready
		if c.InState(StateConnected) {
			log.Logger.Debugf("proxyguard is ready again when the client was already connected")
			return
		}
		log.Logger.Debugf("forwarding proxyguard ready callback to client")
		ready()
	}

	// we set peer IPs to nil here as proxyguard already does a DNS request for us
	err = proxyguard.Client(ck.Context(), listen, tcpsp, peer, nil, -1)
	if err != nil {
		return i18nerr.Wrap(err, "The VPN proxy exited")
	}
	return err
}
