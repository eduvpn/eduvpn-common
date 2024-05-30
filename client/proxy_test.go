package client

import (
	"context"
	"errors"
	"testing"

	"codeberg.org/eduVPN/proxyguard"
)

func TestProxy(t *testing.T) {
	// test race
	p := Proxy{}
	p.NewClient(&proxyguard.Client{})
	go func() {
		// connect to localhost will fail
		// but we don't care about the error
		_ = p.Tunnel(context.Background(), "127.0.0.1")
	}()
	// race!
	_ = p.Cancel()

	// cancel before tunneling
	p.NewClient(&proxyguard.Client{})
	if !errors.Is(p.Cancel(), ErrNoProxyGuardCancel) {
		t.Fatalf("proxyguard cancel err not equal")
	}
	_ = p.Tunnel(context.Background(), "127.0.0.1")
	p.Delete()

	// tunnel without client
	gerr := p.Tunnel(context.Background(), "127.0.0.1")
	if !errors.Is(gerr, ErrNoProxyGuardClient) {
		t.Fatalf("no proxyguard client err not equal")
	}
}
