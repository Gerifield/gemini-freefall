// Package proxy .
package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const (
	headerAPIKey = "x-goog-api-key"

	// TODO: use the original URL maybe with changing the model name only, because this doens't work with streaming, just the generateContents
	// We could maybe use the `:....` ending to support both generateContent and generateContentStream
	// Just look and cut that and use it or default back to generateContent
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

	// Read the request body into a buffer
	var bodyBuffer []byte
	if r.Body != nil {
		// Read the body into the buffer
		var err error
		bodyBuffer, err = io.ReadAll(r.Body)
		if err != nil {
			slog.Error("error reading request body", slog.String("error", err.Error()))
			http.Error(w, "error reading request body", http.StatusInternalServerError)
			return
		}
	}

	slog.Info("proxy configuration found for API key", slog.String("apiKey", apiKey), slog.Any("proxyCfg", proxyCfg))
	for _, c := range proxyCfg {
		backend, err := getBackend(c, l.config)
		if err != nil {
			slog.Error("failed to get backend", slog.String("backend", c), slog.Any("err", err), slog.String("proxyPath", c))

			continue
		}

		// Construct the final URL based on backend type
		var targetURL string
		if backend.Type == "gemini" {
			targetURL = fmt.Sprintf(targetURLPattern, modelName(c))
		} else {
			baseURL := backend.BaseURL
			if baseURL == "" {
				baseURL = l.config.BaseURLs[backend.Type]
			}

			// Append incoming path, unconditionally stripping "/v1" if it's there
			// to avoid duplicating API versions or creating invalid endpoints
			incomingPath := r.URL.Path
			if strings.HasPrefix(incomingPath, "/v1/") {
				incomingPath = strings.TrimPrefix(incomingPath, "/v1")
			}
			targetURL = baseURL + incomingPath
		}

		// Forward the request to the backend
		slog.Info("Forwarding request to backend", slog.String("backend", backend.Name), slog.String("proxyPath", c), slog.String("targetURL", targetURL))
		req, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(bodyBuffer))
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
		if backend.Type == "gemini" {
			req.Header.Set(headerAPIKey, backend.Key)
		} else if backend.Type == "anthropic" {
			req.Header.Set("x-api-key", backend.Key)
			req.Header.Del(headerAPIKey)
			req.Header.Del("Authorization")
		} else {
			req.Header.Set("Authorization", "Bearer "+backend.Key)
			req.Header.Del(headerAPIKey)
		}

		resp, err := l.httpClient.Do(req)
		if err != nil {
			slog.Error("failed to forward request to backend", slog.String("backend", backend.Name), slog.Any("err", err), slog.String("proxyPath", c))

			continue
		}

		if resp.StatusCode != http.StatusOK {
			// Log the response body for debugging purposes
			b, _ := io.ReadAll(resp.Body)
			slog.Error("backend returned non-OK status", slog.String("backend", backend.Name), slog.Int("statusCode", resp.StatusCode), slog.String("proxyPath", c), slog.String("body", string(b)))

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
