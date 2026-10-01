// Command server runs ProfilePulse as a standalone HTTP server. It uses the
// exact same handlers as the Vercel serverless deployment so behaviour is
// identical; only the process model differs.
package main

import (
	"log"
	"net/http"

	"github.com/kirasync2748/profile-pulse/internal/config"
	"github.com/kirasync2748/profile-pulse/internal/server"
)

func main() {
	cfg := config.Load()
	srv := server.New(server.NewStore(cfg))

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.RootHandler)
	mux.HandleFunc("/health", srv.HealthHandler)
	mux.HandleFunc("/api/badge", srv.BadgeHandler)
	mux.HandleFunc("/api/count", srv.CountHandler)

	log.Printf("ProfilePulse listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
