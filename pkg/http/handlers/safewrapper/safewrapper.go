package safewrapper

import (
	"net/http"
	"sync"
)

// SafeWrapper is a http.Handler which wraps another http.Handler which could be
// later replaced by another one.
type SafeWrapper struct {
	handler http.Handler
	mu      sync.RWMutex
}

// New returns a *SafeWrapper initialized with a handler
func New(handler http.Handler) *SafeWrapper {
	h := SafeWrapper{
		handler: handler,
	}

	return &h
}

// ServeHTTP implements http.Handler
func (h *SafeWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	h.handler.ServeHTTP(w, r)
	h.mu.RUnlock()
}

// SwapHandler swaps the handler
func (h *SafeWrapper) SwapHandler(handler http.Handler) {
	h.mu.Lock()
	h.handler = handler
	h.mu.Unlock()
}
