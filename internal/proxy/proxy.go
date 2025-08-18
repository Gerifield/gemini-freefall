// Package proxy .
package proxy

import (
	"fmt"
	"net/http"
)

// Logic .
type Logic struct {
	config *Config
}

// New .
func New(config *Config) *Logic {
	return &Logic{
		config: config,
	}
}

func (l *Logic) handler(w http.ResponseWriter, r *http.Request) {
	// Here you would implement the logic to handle the request
	// For example, you could route requests based on the config
	// or perform some authentication checks.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello from proxy!"))
}

func (l *Logic) ListenAndServe() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", l.config.Config.Port), http.HandlerFunc(l.handler))
}
