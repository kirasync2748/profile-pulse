// Package count is the Vercel serverless entry point for /api/count, which
// returns the current view count as JSON.
package count

import (
	"net/http"

	"github.com/kirasync2748/profile-pulse/internal/config"
	"github.com/kirasync2748/profile-pulse/internal/server"
)

// Handler is mounted by Vercel at /api/count.
func Handler(w http.ResponseWriter, r *http.Request) {
	server.Default(config.Load()).CountHandler(w, r)
}
