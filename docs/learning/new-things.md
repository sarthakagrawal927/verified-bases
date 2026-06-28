# new-things — study queue

Short stubs for non-standard tech in this repo. 3–5 lines each. Fill `Why here:`
yourself after learning; never invent rationale.

## Go on Cloudflare Workers (TinyGo → WebAssembly)
- What: Running Go on Cloudflare Workers by compiling to WebAssembly via TinyGo
- Why here: TBD
- Gotcha (from code): `api/main.go:20-21` — uses `syumai/workers` library; TinyGo compiles to `build/app.wasm`, then deployed via wrangler
- Source: https://github.com/syumai/workers

## Dodo Payments integration (Stripe alternative)
- What: Payment processing via Dodo Payments — international-friendly alternative to Stripe
- Why here: TBD
- Gotcha (from code): `api/handlers/dodo.go:26-32` — base URL switches between `live.dodopayments.com` and `test.dodopayments.com` based on `DODO_ENV` env var
- Source: https://dodopayments.com/

## Astro + Go split architecture with service binding
- What: Frontend is Astro static site on Pages, backend is Go Worker — Pages Functions proxy to Worker via service binding
- Why here: TBD
- Gotcha (from code): `web/functions/api/[[path]].ts:19-33` — Pages Functions proxy to Go Worker via service binding — keeps frontend on single origin (no CORS in prod)
- Source: https://developers.cloudflare.com/pages/functions/bindings/

## Standard Webhooks signature verification
- What: Implementing Standard Webhooks HMAC-SHA256 signature verification manually in Go
- Why here: TBD
- Gotcha (from code): `api/handlers/dodo.go:119-156` — signed payload is `<id>.<timestamp>.<body>`, header carries `v1,base64sig` entries, rejects signatures older than 5 minutes
- Source: https://standardwebhooks.com/

## Turnstile bot verification
- What: Cloudflare Turnstile for bot protection — invisible CAPTCHA alternative
- Why here: TBD
- Gotcha (from code): `api/handlers/turnstile.go` — verifies Turnstile tokens server-side before allowing form submissions; the verification endpoint is `https://challenges.cloudflare.com/turnstile/v0/siteverify`
- Source: https://developers.cloudflare.com/turnstile/

## Service binding pattern (Pages Functions → Workers)
- What: Pages Functions forward requests to Workers via service binding — single origin, no CORS
- Why here: TBD
- Gotcha (from code): `web/functions/api/[[path]].ts:16` — interface `Env { API: { fetch: typeof fetch } }` — service binding named "API" points to `verified-bases-api` Worker
- Source: https://developers.cloudflare.com/workers/runtime-apis/bindings/service-bindings/
