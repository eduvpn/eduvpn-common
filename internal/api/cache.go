package api

import (
	"context"
	"sync"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/api/endpoints"
)

// EndpointCache is a struct that caches well-known API endpoints
type EndpointCache struct {
	lastUpdate map[string]time.Time
	lastEP     map[string]*endpoints.Endpoints
	mu         sync.Mutex
}

// Get() returns a cached or fresh endpoint cache copy
func (ec *EndpointCache) Get(ctx context.Context, wk string) (*endpoints.Endpoints, error) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	// get the last update time
	lu := time.Time{}
	if v, ok := ec.lastUpdate[wk]; ok {
		lu = v
	}

	// if not 10 minutes have passed, return cached copy
	if !lu.IsZero() && !time.Now().After(lu.Add(10*time.Minute)) {
		v, ok := ec.lastEP[wk]
		if ok {
			return v, nil
		}
	}

	// get fresh API endpoints
	ep, err := getEndpoints(ctx, wk)
	if err != nil {
		return nil, err
	}

	// update endpoints
	ec.lastUpdate[wk] = time.Now()
	ec.lastEP[wk] = ep

	return ep, nil
}

var (
	epCache     *EndpointCache
	epCacheOnce sync.Once
)

func GetEndpointCache() *EndpointCache {
	epCacheOnce.Do(func() {
		epCache = &EndpointCache{
			lastUpdate: make(map[string]time.Time),
			lastEP:     make(map[string]*endpoints.Endpoints),
		}
	})

	return epCache
}
