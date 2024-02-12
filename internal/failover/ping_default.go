//go:build !windows

package failover

import (
	"fmt"
	"net"

	"golang.org/x/net/icmp"
)

// NewPinger creates a new pinger with gateway `gateway` and size `size`
func NewPinger(gateway string, size int) (*Pinger, error) {
	l, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return nil, fmt.Errorf("failed creating ping with error: %w", err)
	}
	return &Pinger{
		listener: l,
		buffer:   make([]byte, size-mtuOverhead),
		gateway:  &net.UDPAddr{IP: net.ParseIP(gateway)},
	}, nil
}
