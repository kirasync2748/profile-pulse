// Package health is the Vercel serverless entry point for /health, which
// reports basic process health and does not require Redis.
package health

import (
	"net/http"

	"github.com/kirasync2748/profile-pulse/internal/config"
	"github.com/kirasync2748/profile-pulse/internal/server"
)

// Handler is mounted by Vercel at /api/health (exposed at /health via rewrite).
func Handler(w http.ResponseWriter, r *http.Request) {
	server.Default(config.Load()).HealthHandler(w, r)
}
