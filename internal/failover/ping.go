package failover

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"github.com/go-errors/errors"
)

// mtuOverhead defines the total MTU overhead for an ICMP ECHO message: 20 bytes IP header + 8 bytes ICMP header
var mtuOverhead = 28

type Pinger struct {
	listener net.PacketConn
	buffer   []byte
}

func NewPinger(size int) (*Pinger, error) {
	l, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return nil, errors.WrapPrefix(err, "failed creating ping", 0)
	}
	return &Pinger{listener: l, buffer: make([]byte, size-mtuOverhead)}, nil
}

func (p Pinger) Send(gateway string, seq int) error {
	errorMessage := fmt.Sprintf("failed sending ping, seq %d", seq)
	// Make a new ICMP message
	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: seq,
			Data: p.buffer,
		},
	}
	// Marshal the message to bytes
	b, err := m.Marshal(nil)
	if err != nil {
		return errors.WrapPrefix(err, errorMessage, 0)
	}
	// And send it to the gateway IP!
	_, err = p.listener.WriteTo(b, &net.UDPAddr{IP: net.ParseIP(gateway)})
	if err != nil {
		return errors.WrapPrefix(err, errorMessage, 0)
	}
	return nil
}
