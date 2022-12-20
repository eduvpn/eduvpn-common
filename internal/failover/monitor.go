package failover

import (
	"context"
	"time"

	"github.com/go-errors/errors"
)

// The DroppedConMon is a connection monitor that checks for an increase in rx bytes in certain intervals
type DroppedConMon struct {
	// pInterval means how the interval in which to send pings
	pInterval time.Duration
	// pAlive means how many pings need to be send before checking if the connection is alive
	pAlive int
	// pDropped means how many pings need to be send before checking if the connection is dropped
	pDropped int
	// The function that reads Rx bytes
	// If this function returns an error, the monitor exits
	readRxBytes func() (int64, error)
	// The cancel context
	// This is used to cancel the dropped connection monitor
	cancel context.CancelFunc
}

func NewDroppedMonitor(pingInterval time.Duration, pAlive int, pDropped int, readRxBytes func() (int64, error)) (*DroppedConMon, error) {
	if pAlive >= pDropped {
		return nil, errors.New("pAlive must be smaller than pDropped")
	}
	return &DroppedConMon{pInterval: pingInterval, pAlive: pAlive, pDropped: pDropped, readRxBytes: readRxBytes}, nil
}

// Dropped checks whether or not the connection is 'dropped'
// In other words, it checks if rx bytes has increased
func (m *DroppedConMon) dropped(startBytes int64) (bool, error) {
	b, err := m.readRxBytes()
	if err != nil {
		return false, err
	}
	return b <= startBytes, nil
}

// Start starts ticking every ping interval and check if the connection is dropped or alive
// This does not check Rx bytes every tick, but rather when pAlive or pDropped is reached
// It returns an error if there was an invalid input or a ping was failed to be sent
func (m *DroppedConMon) Start(gateway string, mtuSize int) (bool, error) {
	if mtuSize <= 0 {
		return false, errors.New("invalid mtu size given")
	}

	// Create a context and save the cancel function
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	defer m.cancel()

	// Create a ping struct with our mtu size
	p, err := NewPinger(mtuSize)
	if err != nil {
		return false, err
	}

	// Read the start Rx bytes
	b, err := m.readRxBytes()
	if err != nil {
		return false, err
	}

	// Create a new ticker that executes our ping function every 'interval' seconds
	// It starts immediately and stops when we reach the end
	ticker := time.NewTicker(m.pInterval)
	defer ticker.Stop()

	// Loop until the max drop counter
	// We begin with 1 as this is used as the sequence number for ping
	for s := 1; s <= m.pDropped; s++ {
		// Send a ping and return if an error occurs
		if err := p.Send(gateway, s); err != nil {
			return false, err
		}

		// Early alive check
		// If not dropped, return
		if s == m.pAlive {
			if d, err := m.dropped(b); !d {
				return false, err
			}
		}
		// Wait for the next tick to continue
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return false, errors.New("failover was cancelled")
		}
	}

	// Dropped check if we have not returned early
	return m.dropped(b)
}

// Cancel cancels the dropped connection failover monitor if there is one
func (m *DroppedConMon) Cancel() {
	if m.cancel != nil {
		m.cancel()
	}
}
