// Package badge is the Vercel serverless entry point for /api/badge, an
// explicit SVG badge endpoint equivalent to the root badge path.
package badge

import (
	"net/http"

	"github.com/kirasync2748/profile-pulse/internal/config"
	"github.com/kirasync2748/profile-pulse/internal/server"
)

// Handler is mounted by Vercel at /api/badge.
func Handler(w http.ResponseWriter, r *http.Request) {
	server.Default(config.Load()).BadgeHandler(w, r)
}
