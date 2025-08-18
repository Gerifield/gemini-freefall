// Package proxy .
package proxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const (
	headerAPIKey = "x-goog-api-key"

	// TODO: use the original URL maybe with changing the model name only, because this doens't work with streaming, just the generateContents
	targetURLPattern = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"
)

// Logic .
type Logic struct {
	config     *Config
	httpClient *http.Client
}

// New .
func New(config *Config) *Logic {
	return &Logic{
		config:     config,
		httpClient: &http.Client{}, // TODO: Add Dialer timeout
	}
}

func (l *Logic) handler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received request", slog.String("method", r.Method), slog.String("url", r.URL.String()))
	apiKey := r.Header.Get(headerAPIKey)

	proxyCfg := l.config.Config.Proxy[apiKey]
	if len(proxyCfg) == 0 {
		slog.Error("no proxy configuration found for API key", slog.String("apiKey", apiKey))
		http.Error(w, "no proxy configuration found for API key", http.StatusForbidden)

		return
	}

	slog.Info("proxy configuration found for API key", slog.String("apiKey", apiKey), slog.Any("proxyCfg", proxyCfg))
	for _, c := range proxyCfg {
		backend, err := getBackend(c, l.config)
		if err != nil {
			slog.Error("failed to get backend", slog.String("backend", c), slog.Any("err", err), slog.String("proxyPath", c))

			continue
		}

		// Forward the request to the backend
		slog.Info("Forwarding request to backend", slog.String("backend", backend.Name), slog.String("proxyPath", c))
		req, err := http.NewRequest(r.Method, fmt.Sprintf(targetURLPattern, modelName(c)), r.Body)
		if err != nil {
			slog.Error("failed to create new request", slog.Any("err", err), slog.String("proxyPath", c))
			continue
		}
		// Copy headers from the original request
		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// Re-set the header for the given backend's api key
		req.Header.Set(headerAPIKey, backend.Key)

		resp, err := l.httpClient.Do(req)
		if err != nil {
			slog.Error("failed to forward request to backend", slog.String("backend", backend.Name), slog.Any("err", err), slog.String("proxyPath", c))

			continue
		}

		if resp.StatusCode != http.StatusOK {
			slog.Error("backend returned non-OK status", slog.String("backend", backend.Name), slog.Int("statusCode", resp.StatusCode), slog.String("proxyPath", c))
			_, _ = io.Copy(w, resp.Body)
			_ = resp.Body.Close() // Close the response body to avoid resource leaks

			continue
		}

		size, _ := io.Copy(w, resp.Body)
		slog.Info("successfully forwarded request", slog.Int64("responseSize", size), slog.String("backend", backend.Name), slog.String("proxyPath", c))

		// We are done, exit the handler
		return
	}

	// If we reach here, it means no backend was found or all backends failed
	slog.Error("no valid backend found for API key")
	http.Error(w, "no valid backend found for API key", http.StatusForbidden)
}

func (l *Logic) ListenAndServe() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", l.config.Config.Port), http.HandlerFunc(l.handler))
}
