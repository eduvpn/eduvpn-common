// package proxy is a wrapper around proxyguard that integrates it with eduvpn-common settings
// - leaves out some options not applicable to the common integration, e.g. fwmark
// - integrates with eduvpn-common's logger
// - integrates eduvpn-common's user agent
package proxy

import (
	"context"

	"codeberg.org/eduVPN/proxyguard"

	"github.com/eduvpn/eduvpn-common/i18nerr"
	httpw "github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/log"
)

// Logger is defined here such that we can update the proxyguard logger
type Logger struct{}

// Logf logs a message with parameters
func (l *Logger) Logf(msg string, params ...interface{}) {
	log.Logger.Infof("[Proxyguard] "+msg, params...)
}

// Log logs a message
func (l *Logger) Log(msg string) {
	log.Logger.Infof("[Proxyguard] %s", msg)
}

type Proxy struct {
	proxyguard.Client
}

// NewProxyguard sets up proxyguard for proxied WireGuard connections
func NewProxyguard(ctx context.Context, lp int, tcpsp int, peer string, setupSocket func(fd int)) (*Proxy, error) {
	proxyguard.UpdateLogger(&Logger{})
	proxy := Proxy{
		proxyguard.Client{
			Peer:          peer,
			ListenPort:    lp,
			TCPSourcePort: tcpsp,
			SetupSocket:   setupSocket,
			UserAgent:     httpw.UserAgent,
		},
	}
	err := proxy.Client.SetupDNS(ctx)
	if err != nil {
		return nil, i18nerr.WrapInternal(err, "The ProxyGuard DNS could not be resolved")
	}

	return &proxy, nil
}

func (p *Proxy) Tunnel(ctx context.Context, wglisten int) error {
	err := p.Client.Tunnel(ctx, wglisten)
	if err != nil {
		return i18nerr.WrapInternal(err, "The VPN proxy exited")
	}
	return nil
}
