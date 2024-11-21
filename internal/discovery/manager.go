package discovery

import (
	"context"
	"sync"

	"codeberg.org/eduVPN/eduvpn-common/internal/log"
)

type Manager struct {
	disco *Discovery

	cancel context.CancelFunc
	mu     sync.RWMutex
	wait   sync.WaitGroup
}

func NewManager(disco *Discovery) *Manager {
	return &Manager{disco: disco}
}

func (m *Manager) lock(write bool) {
	if write {
		m.mu.Lock()
		return
	}
	m.mu.RLock()
}

func (m *Manager) unlock(write bool) {
	if write {
		m.mu.Unlock()
		return
	}
	m.mu.RUnlock()
}

func (m *Manager) Discovery(write bool) (*Discovery, func()) {
	if write {
		m.wait.Wait()
	}
	m.lock(write)
	return m.disco, func() {
		m.unlock(write)
	}
}

func (m *Manager) Cancel() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wait.Wait()
}

func (m *Manager) Startup(ctx context.Context, cb func()) {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.wait.Add(1)
	go func() {
		m.lock(false)
		discoCopy, err := m.disco.Copy()
		if err != nil {
			log.Logger.Warningf("internal error, failed to clone discovery, %v", err)
			return
		}
		m.unlock(false)
		// we already log the warning
		discoCopy.Servers(ctx) //nolint:errcheck

		m.lock(true)
		m.disco.UpdateServers(discoCopy)
		m.unlock(true)
		m.wait.Done()

		select {
		case <-ctx.Done():
			return
		default:
			if cb == nil {
				return
			}
			cb()
		}
	}()
}
