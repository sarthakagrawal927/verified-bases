package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/syumai/workers/cloudflare/kv"
)

// allowedOrigins is the list of frontends permitted to call the API.
var allowedOrigins = map[string]bool{
	"https://bases.sarthakagrawal.dev": true,
	"https://verified-bases.pages.dev": true,
	"http://localhost:4321":            true,
	"http://127.0.0.1:4321":            true,
}

// WithCORS sets allowlisted CORS headers on every response.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")
		next.ServeHTTP(w, r)
	})
}

// CORSPreflight ends any OPTIONS request — headers were set in WithCORS.
func CORSPreflight(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// WithRateLimit enforces a sliding-ish per-IP-per-endpoint cap, backed by KV.
//
// Limits are intentionally conservative for a curated marketplace:
//
//	submit, intent, checkout — 20/min per IP
//	webhook                   — unlimited (Dodo retries; rejection breaks fulfillment)
//
// KV is eventually consistent, so this is a defence-in-depth layer, not an
// atomic gate. For higher-volume Phase 2 traffic, swap for Cloudflare's
// Rate Limiting binding (available 2025+) or a Durable Object.
func WithRateLimit(next http.Handler) http.Handler {
	const window = 60 // seconds
	const max = 20

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never rate-limit health checks or the webhook receiver.
		if r.URL.Path == "/api/health" || strings.HasPrefix(r.URL.Path, "/api/webhook/") {
			next.ServeHTTP(w, r)
			return
		}

		ip := clientIP(r)
		key := fmt.Sprintf("rl:%s:%s", ip, r.URL.Path)

		ns, err := kv.NewNamespace("RATELIMIT")
		if err != nil {
			// KV not bound (e.g. tests) — fail open, log via header.
			w.Header().Set("X-RateLimit-Backend", "missing")
			next.ServeHTTP(w, r)
			return
		}

		cur := 0
		if s, err := ns.GetString(key, nil); err == nil && s != "" {
			cur, _ = strconv.Atoi(s)
		}
		if cur >= max {
			w.Header().Set("Retry-After", strconv.Itoa(window))
			writeJSON(w, http.StatusTooManyRequests, `{"ok":false,"error":"rate_limited"}`)
			return
		}
		_ = ns.PutString(key, strconv.Itoa(cur+1), &kv.PutOptions{ExpirationTTL: window})
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max-cur-1))
		next.ServeHTTP(w, r)
	})
}

// clientIP reads the CF-Connecting-IP set by Cloudflare on every Worker
// invocation; falls back to X-Forwarded-For / RemoteAddr.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	return r.RemoteAddr
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func clean(s string) string { return strings.TrimSpace(s) }

// env looks up a Worker env var (vars or secrets).
func env(key string) string { return os.Getenv(key) }
