// Package redis opens configured Redis connections.
package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	redisclient "github.com/redis/go-redis/v9"
)

// Config controls a standalone Redis client. URL takes precedence over the
// individual address and credential fields when provided.
type Config struct {
	URL                   string
	Addr                  string
	Username              string
	Password              string
	DB                    int
	ClientName            string
	Protocol              int
	DialTimeout           time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	PoolSize              int
	MinIdleConns          int
	MaxIdleConns          int
	MaxActiveConns        int
	ConnMaxIdleTime       time.Duration
	ConnMaxLifetime       time.Duration
	TLSConfig             *tls.Config
	ContextTimeoutEnabled bool
}

// DefaultConfig returns defaults suitable for a local Redis instance.
func DefaultConfig() Config {
	return Config{
		Addr:                  "localhost:6379",
		Protocol:              3,
		DialTimeout:           5 * time.Second,
		ReadTimeout:           5 * time.Second,
		WriteTimeout:          5 * time.Second,
		ContextTimeoutEnabled: true,
	}
}

// Open creates a Redis client and verifies the connection with PING.
func Open(ctx context.Context, cfg Config) (*redisclient.Client, error) {
	if ctx == nil {
		return nil, errors.New("redis: context is required")
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}

	options, err := options(cfg)
	if err != nil {
		return nil, err
	}
	client := redisclient.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping %s: %w", options.Addr, err)
	}
	return client, nil
}

func options(cfg Config) (*redisclient.Options, error) {
	var options *redisclient.Options
	if strings.TrimSpace(cfg.URL) != "" {
		parsed, err := redisclient.ParseURL(strings.TrimSpace(cfg.URL))
		if err != nil {
			return nil, fmt.Errorf("redis: parse URL: %w", err)
		}
		options = parsed
	} else {
		addr := strings.TrimSpace(cfg.Addr)
		if addr == "" {
			return nil, errors.New("redis: address or URL is required")
		}
		options = &redisclient.Options{
			Addr:      addr,
			Username:  cfg.Username,
			Password:  cfg.Password,
			DB:        cfg.DB,
			TLSConfig: cfg.TLSConfig,
		}
	}

	options.ClientName = cfg.ClientName
	if cfg.Protocol != 0 {
		options.Protocol = cfg.Protocol
	}
	if cfg.DialTimeout != 0 {
		options.DialTimeout = cfg.DialTimeout
	}
	if cfg.ReadTimeout != 0 {
		options.ReadTimeout = cfg.ReadTimeout
	}
	if cfg.WriteTimeout != 0 {
		options.WriteTimeout = cfg.WriteTimeout
	}
	if cfg.PoolSize > 0 {
		options.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		options.MinIdleConns = cfg.MinIdleConns
	}
	if cfg.MaxIdleConns > 0 {
		options.MaxIdleConns = cfg.MaxIdleConns
	}
	if cfg.MaxActiveConns > 0 {
		options.MaxActiveConns = cfg.MaxActiveConns
	}
	if cfg.ConnMaxIdleTime != 0 {
		options.ConnMaxIdleTime = cfg.ConnMaxIdleTime
	}
	if cfg.ConnMaxLifetime != 0 {
		options.ConnMaxLifetime = cfg.ConnMaxLifetime
	}
	options.ContextTimeoutEnabled = cfg.ContextTimeoutEnabled
	return options, nil
}

func validate(cfg Config) error {
	if cfg.DB < 0 {
		return errors.New("redis: DB cannot be negative")
	}
	if cfg.Protocol != 0 && cfg.Protocol != 2 && cfg.Protocol != 3 {
		return errors.New("redis: protocol must be 2 or 3")
	}
	if cfg.PoolSize < 0 || cfg.MinIdleConns < 0 || cfg.MaxIdleConns < 0 || cfg.MaxActiveConns < 0 {
		return errors.New("redis: pool values cannot be negative")
	}
	if cfg.PoolSize > 0 && cfg.MinIdleConns > cfg.PoolSize {
		return errors.New("redis: MinIdleConns cannot exceed PoolSize")
	}
	return nil
}
