# verified-bases-api

Go backend for the Verified Software Bases marketplace, running on
**Cloudflare Workers** via **TinyGo** → WASM, using
[`syumai/workers`](https://github.com/syumai/workers) for the
`fetch` ⇄ `net/http` bridge.

## Why this stack

- **Go** for the BE (per project decision)
- **TinyGo** because stdlib Go WASM binaries are 20 MB+ and blow the Worker
  3 MB-gzipped limit; TinyGo lands around 1 MB.
- **Cloudflare Workers** so the API runs on the same edge as the Pages site
  with zero cold-start matter.

## Endpoints

| Method | Path           | Purpose                                                 |
| ------ | -------------- | ------------------------------------------------------- |
| GET    | `/api/health`  | Liveness + phase label                                   |
| POST   | `/api/submit`  | Creator submission form (forwards to admin inbox)        |
| POST   | `/api/intent`  | Buyer tier-intent capture (forwards to admin inbox)      |

Phase 1 is **manual fulfillment** — both POST endpoints just relay to a
Discord/Slack webhook (`ADMIN_INBOX_WEBHOOK` secret). Phase 2 swaps `/intent`
for a real Stripe Checkout session.

## Prerequisites

```bash
# Install Go 1.22+
brew install go

# Install TinyGo (required — stdlib Go WASM is too big for Workers)
brew tap tinygo-org/tools
brew install tinygo

# wrangler is invoked via npx, so npm/node already on the box is enough
node --version    # >= 18
```

## Local dev

```bash
make build                                    # tinygo → build/app.wasm
make dev                                      # wrangler dev on :8787
# In another tab:
curl http://localhost:8787/api/health         # {"ok":true,...}
```

For local secrets, create `api/.dev.vars`:

```
ADMIN_INBOX_WEBHOOK="https://discord.com/api/webhooks/…"
```

## Deploy

```bash
# One-time, per environment
wrangler secret put ADMIN_INBOX_WEBHOOK

# Each deploy
make deploy
```

## Wire-up with the Pages site

Two clean options, both supported by Cloudflare:

1. **Pages Functions proxy** (recommended) — add `web/functions/api/[[path]].ts`
   that forwards `/api/*` to the Worker. The site stays a single origin.
2. **Workers route** on the same subdomain — bind the Worker to
   `bases.sarthakagrawal.dev/api/*` in the Worker's `wrangler.jsonc`.

For Phase 1, Option 1 is simpler.

## Layout

```
api/
├── main.go                 — router + entrypoint
├── handlers/
│   ├── middleware.go       — CORS + JSON helpers
│   ├── health.go
│   ├── submit.go           — creator form
│   ├── intent.go           — buyer tier intent
│   └── inbox.go            — outbound webhook forwarder
├── go.mod
├── Makefile                — build / dev / deploy targets
├── wrangler.jsonc          — Worker config (compatibility_date, vars, build)
└── README.md (this file)
```

## Notes / gotchas

- TinyGo's `net/http` support is intentionally minimal — `syumai/workers`
  bridges what's missing. Avoid goroutines, `reflect`, and anything
  needing cgo.
- Body reads must `defer r.Body.Close()` even in single-shot workers.
- Worker secrets do **not** appear in `wrangler.jsonc`; set them via
  `wrangler secret put`.
