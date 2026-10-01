// Package server contains the HTTP handlers shared by the Vercel serverless
// entry points (api/) and the standalone development server (cmd/server).
//
// All untrusted input is validated before it reaches the counter or the SVG
// renderer, and every dynamic value placed into a response is escaped. Redis
// failures are mapped to a generic "Counter unavailable" error badge and a 5xx
// status without leaking internal details.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/kirasync2748/profile-pulse/internal/badge"
	"github.com/kirasync2748/profile-pulse/internal/config"
	"github.com/kirasync2748/profile-pulse/internal/counter"
	"github.com/kirasync2748/profile-pulse/internal/validation"
)

// Server holds the dependencies shared across handlers.
type Server struct {
	store counter.Store
}

// New creates a Server backed by the given counter store.
func New(store counter.Store) *Server {
	return &Server{store: store}
}

// defaultServer is lazily initialized for the Vercel entry points, which each
// run as isolated serverless invocations.
var (
	defaultServer     *Server
	defaultServerOnce sync.Once
)

// Default returns a Server configured from environment variables. It selects
// the Upstash REST store when credentials are present, a local Redis store
// when a Redis URL is present, and a no-op store otherwise. The result is
// cached for the lifetime of the process.
func Default(cfg config.Config) *Server {
	defaultServerOnce.Do(func() {
		defaultServer = New(NewStore(cfg))
	})
	return defaultServer
}

// NewStore selects a counter.Store from the configuration.
func NewStore(cfg config.Config) counter.Store {
	switch {
	case cfg.HasUpstash():
		return counter.NewUpstashStore(cfg.UpstashRedisRestURL, cfg.UpstashRedisRestToken)
	case cfg.HasRedis():
		s, err := counter.NewLocalStore(cfg.RedisURL)
		if err != nil {
			return counter.NoopStore{}
		}
		return s
	default:
		return counter.NoopStore{}
	}
}

// --- headers ----------------------------------------------------------------

func setSVGHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "image/svg+xml; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("Pragma", "no-cache")
	h.Set("Access-Control-Allow-Origin", "*")
}

func setJSONHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("Access-Control-Allow-Origin", "*")
}

// --- error helpers ----------------------------------------------------------

// writeErrorBadge writes a valid SVG badge describing an error condition.
func writeErrorBadge(w http.ResponseWriter, status int, message string) {
	setSVGHeaders(w)
	w.WriteHeader(status)
	_, _ = w.Write(badge.Render(badge.NewParams("error", message, "e05d44", "flat")))
}

// writeJSONError writes a JSON error object with the given HTTP status.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	setJSONHeaders(w)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// --- handlers ---------------------------------------------------------------

// RootHandler returns a simple JSON health payload at the root path.
func (s *Server) RootHandler(w http.ResponseWriter, r *http.Request) {
	setJSONHeaders(w)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HealthHandler reports basic process health. It does NOT require Redis.
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	setJSONHeaders(w)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// BadgeHandler validates inputs, atomically increments the counter, and
// returns an SVG badge. On validation failure it returns an error badge with
// a 4xx status; on counter failure it returns an error badge with a 5xx
// status.
func (s *Server) BadgeHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	username, err := validation.ValidateUsername(q.Get("username"))
	if err != nil {
		writeErrorBadge(w, http.StatusBadRequest, err.Error())
		return
	}
	color, err := validation.ValidateColor(q.Get("color"))
	if err != nil {
		writeErrorBadge(w, http.StatusBadRequest, "invalid color")
		return
	}
	style, err := validation.ValidateStyle(q.Get("style"))
	if err != nil {
		writeErrorBadge(w, http.StatusBadRequest, "invalid style")
		return
	}
	label, err := validation.ValidateLabel(q.Get("label"))
	if err != nil {
		writeErrorBadge(w, http.StatusBadRequest, "invalid label")
		return
	}
	base, err := config.ParseBase(q.Get("base"))
	if err != nil {
		writeErrorBadge(w, http.StatusBadRequest, "invalid base")
		return
	}
	abbreviated := validation.ParseBool(q.Get("abbreviated"))

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	count, err := s.store.Incr(ctx, validation.RedisKey(username))
	if err != nil {
		writeErrorBadge(w, http.StatusServiceUnavailable, "Counter unavailable")
		return
	}

	value := badge.DisplayValue(count, base, abbreviated)
	setSVGHeaders(w)
	_, _ = w.Write(badge.Render(badge.NewParams(label, value, color, style)))
}

// CountHandler returns the current view count as JSON. Like the badge
// endpoint it increments the counter on each call.
func (s *Server) CountHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	username, err := validation.ValidateUsername(q.Get("username"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	base, err := config.ParseBase(q.Get("base"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid base")
		return
	}
	abbreviated := validation.ParseBool(q.Get("abbreviated"))

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	count, err := s.store.Incr(ctx, validation.RedisKey(username))
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "counter unavailable")
		return
	}

	setJSONHeaders(w)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"username": username,
		"views":    count + base,
		"raw":      count,
		"display":  badge.DisplayValue(count, base, abbreviated),
	})
}
