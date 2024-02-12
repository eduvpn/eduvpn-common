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

func (c *Client) StartProxyguard(ck *cookie.Cookie, listen string, tcpsp int, peer string) error {
// StartProxyguard starts proxyguard for proxied WireGuard connections
	var err error
	proxyguard.UpdateLogger(&ProxyLogger{})
	err = proxyguard.Client(ck.Context(), listen, tcpsp, peer, -1)
	if err != nil {
		return i18nerr.Wrap(err, "The VPN proxy exited")
	}
	return err
}
