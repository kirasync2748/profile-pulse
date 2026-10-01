// Package config loads runtime configuration from environment variables.
//
// ProfilePulse is designed for Vercel serverless deployment backed by Upstash
// Redis. In production the Upstash REST credentials are provided through
// environment variables. For local development a standard Redis URL may be
// used instead; the server selects the appropriate counter store based on
// which variables are present (see internal/counter).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration for the service.
type Config struct {
	// Upstash Redis REST API credentials (production / Vercel).
	UpstashRedisRestURL   string
	UpstashRedisRestToken string

	// Standard Redis URL for local development (e.g. redis://localhost:6379).
	// When set and no Upstash credentials are present, the local Redis store is
	// used.
	RedisURL string

	// Addr is the address the HTTP server listens on.
	Addr string
}

// Load reads configuration from environment variables, applying sensible
// defaults for local development.
func Load() Config {
	return Config{
		UpstashRedisRestURL:   strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_URL")),
		UpstashRedisRestToken: strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_TOKEN")),
		RedisURL:              strings.TrimSpace(os.Getenv("REDIS_URL")),
		Addr:                  loadAddr(),
	}
}

// loadAddr resolves the listen address. In production (Vercel Go Framework
// preset) the platform sets PORT as a bare port number (e.g. "3000"); the
// server must listen on it. In local development ADDR is used as a full
// address (e.g. ":3000"). Falls back to ":3000".
func loadAddr() string {
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		if strings.HasPrefix(port, ":") {
			return port
		}
		return ":" + port
	}
	return envOr("ADDR", ":3000")
}

// HasUpstash reports whether Upstash REST credentials are configured.
func (c Config) HasUpstash() bool {
	return c.UpstashRedisRestURL != "" && c.UpstashRedisRestToken != ""
}

// HasRedis reports whether a standard Redis URL is configured.
func (c Config) HasRedis() bool {
	return c.RedisURL != ""
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// ParseBase parses a base offset query parameter. It must be a non-negative
// integer. An empty value yields 0. Negative or excessively large values are
// rejected.
func ParseBase(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid base: must be an integer")
	}
	if n < 0 {
		return 0, fmt.Errorf("invalid base: must be non-negative")
	}
	if n > 1_000_000_000 {
		return 0, fmt.Errorf("invalid base: too large")
	}
	return n, nil
}
