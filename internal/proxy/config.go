package proxy

import (
	"errors"
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
	Backend []Backend `yaml:"backend"`
	Config  struct {
		Port  int                 `yaml:"port"`
		Proxy map[string][]string `yaml:"proxy"`
	} `yaml:"config"`
}

type Backend struct {
	Name   string   `yaml:"name"`
	Key    string   `yaml:"key"`
	Models []string `yaml:"models"`
}

// LoadConfig .
func LoadConfig(fname string) (*Config, error) {
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
