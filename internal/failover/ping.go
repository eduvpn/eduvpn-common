package failover

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// mtuOverhead defines the total MTU overhead for an ICMP ECHO message: 20 bytes IP header + 8 bytes ICMP header
var mtuOverhead = 28

type Pinger struct {
	listener net.PacketConn
	buffer   []byte
	gateway  net.Addr
}

func (p Pinger) Read(deadline time.Time) error {
	// First set the deadline to read
	err := p.listener.SetReadDeadline(deadline)
	if err != nil {
		return err
	}

	r := make([]byte, 1500)
	n, _, err := p.listener.ReadFrom(r)
	if err != nil {
		return err
	}
	got, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), r[:n])
	if err != nil {
		return err
	}
	switch got.Type {
	case ipv4.ICMPTypeEchoReply:
		return nil
	default:
		return fmt.Errorf("not a ping echo reply, got: %+v", got)
	}
}

func (p Pinger) Send(seq int) error {
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
		return fmt.Errorf("%s with error: %w", errorMessage, err)
	}
	// And send it to the gateway IP!
	_, err = p.listener.WriteTo(b, p.gateway)
	if err != nil {
		return fmt.Errorf("%s with error: %w", errorMessage, err)
	}

	return nil
}
