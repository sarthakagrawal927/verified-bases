// verified-bases API — Cloudflare Worker (Go via TinyGo + syumai/workers).
//
// Endpoints
//
//	GET  /api/health              — liveness + phase
//	POST /api/submit              — creator submission
//	POST /api/intent              — buyer interest capture (no payment)
//	POST /api/checkout            — start a Dodo Payments checkout session
//	POST /api/webhook/dodo        — Dodo webhook receiver (Standard Webhooks signing)
//
// Build:  make build      (TinyGo → build/app.wasm)
// Dev:    make dev        (wrangler dev, local D1, local KV)
// Deploy: make deploy
package main

import (
	"net/http"

	"github.com/sarthakagrawal927/verified-bases/api/handlers"
	"github.com/syumai/workers"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", handlers.Health)

	mux.HandleFunc("POST /api/submit", handlers.Submit)
	mux.HandleFunc("POST /api/intent", handlers.Intent)
	mux.HandleFunc("POST /api/checkout", handlers.Checkout)
	mux.HandleFunc("POST /api/webhook/dodo", handlers.DodoWebhook)
	mux.HandleFunc("GET /api/download", handlers.Download)

	mux.HandleFunc("OPTIONS /api/", handlers.CORSPreflight)

	// Middleware order matters: CORS outermost → rate-limit → handler.
	workers.Serve(handlers.WithCORS(handlers.WithRateLimit(mux)))
}
