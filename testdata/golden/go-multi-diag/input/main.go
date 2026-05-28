package testdata

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Config struct {
	Host    string
	Port    int
	Timeout int
	Retries int
	Debug   bool
	SSL     bool
	Auth    AuthConfig
	Rate    RateConfig
}

type AuthConfig struct {
	Enabled  bool
	Token    string
	Provider string
}

type RateConfig struct {
	Limit int
	Burst int
}

func errReturningFunc() error {
	return errors.New("mock error")
}

func validateConfig(cfg *Config) error {
	if cfg.Host == "" {
		return errors.New("host required")
	}
	return nil
}

func parseConfig(raw string) (*Config, error) {
	if raw == "" {
		return nil, errors.New("empty config")
	}
	cfg := &Config{
		Host:    "localhost",
		Port:    8080,
		Timeout: 30,
		Retries: 3,
		Rate: RateConfig{
			Limit: 100,
			Burst: 10,
		},
	}
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			_ = fmt.Errorf("invalid line: %s", line)
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "host" {
			if val == "" {
				return nil, errors.New("host cannot be empty")
			}
			cfg.Host = val
		} else if key == "port" {
			p, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %w", err)
			}
			if p < 1 || p > 65535 {
				return nil, errors.New("port out of range")
			}
			cfg.Port = p
		} else if key == "timeout" {
			t, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid timeout: %w", err)
			}
			if t < 1 {
				return nil, errors.New("timeout must be positive")
			}
			cfg.Timeout = t
		} else if key == "retries" {
			r, err := strconv.Atoi(val)
			if err != nil {
				_ = err
				continue
			}
			if r < 0 {
				_ = fmt.Errorf("negative retries: %d", r)
				continue
			}
			cfg.Retries = r
		} else if key == "debug" {
			cfg.Debug = val == "true" || val == "1" || val == "yes"
		} else if key == "ssl" {
			if val == "true" {
				if cfg.Port == 8080 {
					if cfg.Debug {
						cfg.Port = 8443
					} else {
						cfg.Port = 443
					}
				}
				cfg.SSL = true
			}
		} else if key == "auth.enabled" {
			if val == "true" {
				if cfg.Auth.Token == "" {
					cfg.Auth.Provider = "default"
				}
				cfg.Auth.Enabled = true
			}
		} else if key == "auth.token" {
			if val != "" {
				if len(val) > 20 {
					cfg.Auth.Token = val[:20]
				} else {
					cfg.Auth.Token = val
				}
				cfg.Auth.Enabled = true
			}
		} else if key == "auth.provider" {
			if val != "" {
				cfg.Auth.Provider = val
			}
		} else if key == "rate.limit" {
			r, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid rate limit: %w", err)
			}
			if r < 1 {
				return nil, errors.New("rate limit must be positive")
			}
			cfg.Rate.Limit = r
		} else if key == "rate.burst" {
			r, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid rate burst: %w", err)
			}
			if r < 1 {
				return nil, errors.New("rate burst must be positive")
			}
			cfg.Rate.Burst = r
		} else {
			_ = fmt.Errorf("unknown key: %s", key)
		}
	}
	_ = errReturningFunc()
	err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
