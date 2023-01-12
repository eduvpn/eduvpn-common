package failover

import (
	"time"

	"github.com/eduvpn/eduvpn-common/internal/log"
)

const (
	// Send a ping every 2 seconds to the gateway
	pInterval time.Duration = 2 * time.Second

	// pDropped is how many pings we need to have sent to check if the connection is dropped
	pDropped int = 5
)

// New creates a failover monitor for the gateway and the rx bytes function reader
// This is a simple wrapper over `NewDroppedMonitor` to create one with the default settings
// If this function returns True, the connection is dropped. False means it has exited and we don't know for sure if it's dropped or not
func New(readRxBytes func() (int64, error), logger log.FileLogger) *DroppedConMon {
	return NewDroppedMonitor(pInterval, pDropped, readRxBytes, logger)
}
