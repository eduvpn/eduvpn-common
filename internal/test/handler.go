package test

import (
	"net/http"
	"sync"
)

// HandlerSet is a struct with a mutex that allows us to swap handlers while a test server is running
type HandlerSet struct {
	mu sync.Mutex
	handler http.Handler
}

func (hs *HandlerSet) SetHandler(handler http.Handler) {
	hs.mu.Lock()
	hs.handler = handler
	hs.mu.Unlock()
}

func (hs *HandlerSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hs.mu.Lock()
	handler := hs.handler
	hs.mu.Unlock()
	handler.ServeHTTP(w, r)
}
