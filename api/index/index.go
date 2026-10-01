// Package index is the Vercel serverless entry point for the root path. It
// returns a JSON health payload: {"status":"ok"}. The vercel.json rewrite
// routes GET / here. The actual logic lives in internal/server so it is
// shared with the standalone development server and fully tested.
package index

import (
	"net/http"

	"github.com/kirasync2748/profile-pulse/internal/config"
	"github.com/kirasync2748/profile-pulse/internal/server"
)

// Handler is mounted by Vercel at /api/index.
func Handler(w http.ResponseWriter, r *http.Request) {
	server.Default(config.Load()).RootHandler(w, r)
}
