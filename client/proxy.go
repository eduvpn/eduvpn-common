package client

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"codeberg.org/eduVPN/proxyguard"

	"github.com/eduvpn/eduvpn-common/i18nerr"
	httpw "github.com/eduvpn/eduvpn-common/internal/http"
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

// Proxy is a wrapper around ProxyGuard
// that has the client
// and a cancel for cancellation by common
// and a mutex to protect against race conditions
type Proxy struct {
	c      *proxyguard.Client
	mu     sync.Mutex
	cancel context.CancelFunc
}

// NewClient creates a new ProxyGuard wrapper from client `c`
func (p *Proxy) NewClient(c *proxyguard.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.c = c
}

// Delete sets the inner client to nil
func (p *Proxy) Delete() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.c = nil
}

// ErrNoProxyGuardCancel indicates that no ProxyGuard cancel function
// was ever defined. You probably forgot to call `Tunnel`
var ErrNoProxyGuardCancel = errors.New("no ProxyGuard cancel function")

// Cancel cancels a running ProxyGuard tunnel
// it returns an error if it cannot be canceled
func (p *Proxy) Cancel() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel == nil {
		return ErrNoProxyGuardCancel
	}
	p.cancel()
	p.cancel = nil
	return nil
}

// ErrNoProxyGuardClient is an error that is returned when no ProxyGuard client is created
var ErrNoProxyGuardClient = errors.New("no ProxyGuard client created")

// Tunnel is a wrapper around ProxyGuard tunnel that
// that creates a new context that can be canceled
func (p *Proxy) Tunnel(ctx context.Context, peer string) error {
	p.mu.Lock()
	if p.c == nil {
		p.mu.Unlock()
		return ErrNoProxyGuardClient
	}
	cctx, cf := context.WithCancel(ctx)
	p.cancel = cf
	client := *p.c
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.cancel = nil
		p.mu.Unlock()
	}()
	// we set peer IPs to nil here as proxyguard already does a DNS request for us
	return client.Tunnel(cctx, peer, nil)
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
		UserAgent: httpw.UserAgent,
		Ready:     ready,
	}

	c.proxy.NewClient(&proxyc)
	defer c.proxy.Delete()
	err = c.proxy.Tunnel(ck.Context(), peer)
	if err != nil {
		return i18nerr.WrapInternal(err, "The VPN proxy exited")
	}
	return err
}
