package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	os.Unsetenv("UPSTASH_REDIS_REST_URL")
	os.Unsetenv("UPSTASH_REDIS_REST_TOKEN")
	os.Unsetenv("REDIS_URL")
	os.Unsetenv("ADDR")
	os.Unsetenv("PORT")

	cfg := Load()
	if cfg.Addr != ":3000" {
		t.Errorf("Addr = %q, want :3000", cfg.Addr)
	}
	if cfg.HasUpstash() {
		t.Error("HasUpstash should be false without credentials")
	}
	if cfg.HasRedis() {
		t.Error("HasRedis should be false without URL")
	}
}

func TestLoadPortTakesPrecedence(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("ADDR", ":9090")

	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080 (PORT should take precedence)", cfg.Addr)
	}
}

func TestLoadPortWithColon(t *testing.T) {
	t.Setenv("PORT", ":7070")

	cfg := Load()
	if cfg.Addr != ":7070" {
		t.Errorf("Addr = %q, want :7070", cfg.Addr)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("UPSTASH_REDIS_REST_URL", "https://example.upstash.io")
	t.Setenv("UPSTASH_REDIS_REST_TOKEN", "secret-token")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("ADDR", ":8080")
	os.Unsetenv("PORT")

	cfg := Load()
	if !cfg.HasUpstash() {
		t.Error("HasUpstash should be true with credentials")
	}
	if !cfg.HasRedis() {
		t.Error("HasRedis should be true with URL")
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.Addr)
	}
}

func TestHasUpstashRequiresBoth(t *testing.T) {
	t.Setenv("UPSTASH_REDIS_REST_URL", "https://example.upstash.io")
	t.Setenv("UPSTASH_REDIS_REST_TOKEN", "")
	cfg := Load()
	if cfg.HasUpstash() {
		t.Error("HasUpstash should be false when token is empty")
	}
}

func TestParseBase(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"1000", 1000, false},
		{"999999999", 999999999, false},
		{"-1", 0, true},
		{"abc", 0, true},
		{"1000000001", 0, true},
		{"1.5", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseBase(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseBase(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseBase(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
