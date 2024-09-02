package failover

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/test"
)

// mockedPinger is a ping sender that always returns nil for sending
// but returns EOF for reading
type mockedPinger struct{}

func (mp *mockedPinger) Read(_ time.Time) error {
	return io.EOF
}

func (mp *mockedPinger) Send(_ int) error {
	return nil
}

func TestMonitor(t *testing.T) {
	cases := []struct {
		interval        time.Duration
		pDropped        int
		readRxBytes     func() (int64, error)
		gateway         string
		mtuSize         int
		disableDefaults bool
		mockedPinger    func(gateway string, mtu int) (sender, error)
		wantDropped     bool
		wantErr         string
	}{
		{
			mtuSize:     1,
			wantDropped: false,
			wantErr:     "invalid MTU size given, MTU has to be at least: 28 bytes",
		},
		{
			readRxBytes: func() (int64, error) {
				return 0, errors.New("error test")
			},
			wantDropped: false,
			wantErr:     "error test",
		},
		// default case, not dropped
		{},
		// readRxBytes always returns 0
		// still we do not want a drop because we get a pong from 127.0.0.1
		{
			readRxBytes: func() (int64, error) {
				return 0, nil
			},
			wantDropped: false,
		},
		// readRxBytes always returns 0
		// we want dropped as the mock pinger does nothing
		{
			readRxBytes: func() (int64, error) {
				return 0, nil
			},
			gateway: "127.0.0.1",
			mockedPinger: func(_ string, _ int) (sender, error) {
				return &mockedPinger{}, nil
			},
			wantDropped: true,
		},
	}

	for _, c := range cases {
		var counter int64
		// some defaults
		if c.interval == 0 {
			c.interval = 2 * time.Second
		}
		if c.pDropped == 0 {
			c.pDropped = 5
		}
		if c.gateway == "" {
			c.gateway = "127.0.0.1"
		}
		if c.mtuSize == 0 {
			c.mtuSize = 28
		}
		if c.readRxBytes == nil {
			c.readRxBytes = func() (int64, error) {
				defer func() {
					counter++
				}()
				return counter, nil
			}
		}
		dcm := NewDroppedMonitor(c.interval, c.pDropped, c.readRxBytes)
		if c.mockedPinger != nil {
			dcm.newPinger = c.mockedPinger
		}
		dropped, err := dcm.Start(context.Background(), c.gateway, c.mtuSize)
		if dropped != c.wantDropped {
			t.Fatalf("dropped is not equal to want dropped, got: %v, want: %v", dropped, c.wantDropped)
		}
		test.AssertError(t, err, c.wantErr)
	}
}
