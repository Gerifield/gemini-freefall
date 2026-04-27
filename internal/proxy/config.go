package proxy

import (
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

var (
	ErrInvalidPort    = errors.New("invalid port number")
	ErrNoBackends     = errors.New("no backends configured")
	ErrInvalidBackend = errors.New("invalid backend configuration")
	ErrInvalidProxy   = errors.New("invalid proxy connection configuration")
	ErrNoModels       = errors.New("no models configured for backend")
)

type Config struct {
	BaseURLs map[string]string `yaml:"base_urls"`
	Backend  []Backend         `yaml:"backend"`
	Config   struct {
		Port  int                 `yaml:"port"`
		Proxy map[string][]string `yaml:"proxy"`
	} `yaml:"config"`
}

type Backend struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	BaseURL string   `yaml:"base_url"`
	Key     string   `yaml:"key"`
	Models  []string `yaml:"models"`
}

// LoadConfig .
func LoadConfig(fname string, openaiMode bool) (*Config, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	var cfg Config
	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return nil, err
	}

	if cfg.BaseURLs == nil {
		cfg.BaseURLs = make(map[string]string)
	}
	if cfg.BaseURLs["openai"] == "" {
		cfg.BaseURLs["openai"] = "https://api.openai.com/v1"
	}
	if cfg.BaseURLs["anthropic"] == "" {
		cfg.BaseURLs["anthropic"] = "https://api.anthropic.com/v1"
	}
	if cfg.BaseURLs["gemini_openai"] == "" {
		cfg.BaseURLs["gemini_openai"] = "https://generativelanguage.googleapis.com/v1beta/openai"
	}

	var filteredBackends []Backend
	for i, b := range cfg.Backend {
		if b.Type == "" {
			b.Type = "gemini"
		}
		cfg.Backend[i] = b

		if openaiMode {
			if b.Type == "gemini" {
				// We drop gemini backends in openai mode
				slog.Warn("dropping gemini backend in openai mode", slog.String("backend", b.Name))
				continue
			}
		} else {
			if b.Type != "gemini" {
				// We drop non-gemini backends in non-openai mode
				slog.Warn("dropping non-gemini backend in default (gemini) mode", slog.String("backend", b.Name), slog.String("type", b.Type))
				continue
			}
		}
		filteredBackends = append(filteredBackends, b)
	}
	cfg.Backend = filteredBackends

	// Drop proxy routes that refer to dropped backends
	filteredProxy := make(map[string][]string)
	for pName, pRoutes := range cfg.Config.Proxy {
		var validRoutes []string
		for _, route := range pRoutes {
			parts := strings.SplitN(route, ".", 2)
			if len(parts) >= 1 {
				found := false
				for _, b := range cfg.Backend {
					if b.Name == parts[0] {
						found = true
						break
					}
				}
				if found {
					validRoutes = append(validRoutes, route)
				}
			}
		}
		if len(validRoutes) > 0 {
			filteredProxy[pName] = validRoutes
		}
	}
	cfg.Config.Proxy = filteredProxy

	return &cfg, checkConfig(&cfg)
}

func checkConfig(cfg *Config) error {
	if cfg.Config.Port == 0 {
		return ErrInvalidPort
	}

	if len(cfg.Backend) == 0 {
		return ErrNoBackends
	}

	for _, b := range cfg.Backend {
		if b.Name == "" || b.Key == "" {
			return ErrInvalidBackend
		}

		if b.Type != "gemini" && b.BaseURL == "" && cfg.BaseURLs[b.Type] == "" {
			return ErrInvalidBackend
		}

		if len(b.Models) == 0 {
			return ErrNoModels
		}
	}

	for _, p := range cfg.Config.Proxy {
		if len(p) == 0 {
			return ErrInvalidProxy
		}

		for _, proxyPath := range p {
			if !isValidBackend(proxyPath, cfg) {
				return ErrInvalidBackend
			}
		}
	}

	return nil
}

func isValidBackend(proxyPath string, cfg *Config) bool {
	_, err := getBackend(proxyPath, cfg)

	return err == nil
}

func modelName(proxyPath string) string {
	parts := strings.SplitN(proxyPath, ".", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func getBackend(proxyPath string, cfg *Config) (Backend, error) {
	parts := strings.SplitN(proxyPath, ".", 2)
	if len(parts) < 2 {
		return Backend{}, ErrInvalidBackend
	}

	for _, backend := range cfg.Backend {
		if backend.Name == parts[0] {
			for _, model := range backend.Models {
				if model == parts[1] {
					return backend, nil
				}
			}
		}
	}

	return Backend{}, ErrInvalidBackend
}
