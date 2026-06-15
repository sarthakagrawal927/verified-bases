# Deploy — Verified Software Bases

> Target: `bases.sarthakagrawal.dev` on Cloudflare (Pages + Workers).

This is a step-by-step run-the-first-time guide. Subsequent deploys are
either `git push` (Pages auto-builds) or `make deploy` (Worker).

## Architecture in one diagram

```
            bases.sarthakagrawal.dev (DNS in Cloudflare)
                           │
                           ▼
        ┌─────────────────────────────────────────┐
        │  Cloudflare Pages  (verified-bases-web) │
        │                                         │
        │   /  → static Astro (web/dist)          │
        │   /api/*  →  Pages Function proxy       │
        └────────────────────┬────────────────────┘
                             │ service binding "API"
                             ▼
        ┌─────────────────────────────────────────┐
        │  Cloudflare Worker (verified-bases-api) │
        │   Go via TinyGo + syumai/workers        │
        │                                         │
        │   bindings:                             │
        │     DB        → D1 (verified-bases-db)  │
        │     RATELIMIT → KV namespace            │
        │   secrets:                              │
        │     DODO_API_KEY, DODO_WEBHOOK_SECRET   │
        │     RESEND_API_KEY                      │
        │     DODO_PRODUCT_<slug>_<tier> (per Base│
        │                            tier in use) │
        └─────────────────────────────────────────┘
```

## 0. One-time prerequisites

```bash
# Local
brew install tinygo node go
node --version    # >=18
go version        # >=1.22
tinygo version    # >=0.34

# Cloudflare account
npx wrangler@latest login
```

## 1. Provision Cloudflare resources

```bash
cd api

# Create D1 database, copy the printed database_id into wrangler.jsonc.
npx wrangler d1 create verified-bases-db
# ⇒ database_id = "...."  ← paste into api/wrangler.jsonc

# Apply schema to remote D1.
make migrate-remote

# Create KV namespace for rate-limit counters; paste id into wrangler.jsonc.
npx wrangler kv:namespace create RATELIMIT

# Create R2 bucket for deliverable files (Use-It / Own-It artifacts).
npx wrangler r2 bucket create verified-bases-delivery

# Create a Turnstile site (Cloudflare dashboard → Turnstile → Add site).
# Domain: bases.sarthakagrawal.dev (+ localhost for dev). Mode: Managed.
# Copy the site key (public) and secret key (private).
```

## 2. Set Worker secrets

```bash
cd api
npx wrangler secret put DODO_API_KEY           # test or live, must match DODO_ENV var
npx wrangler secret put DODO_WEBHOOK_SECRET    # from Dodo dashboard → Developer → Webhooks
npx wrangler secret put RESEND_API_KEY         # https://resend.com → API keys
npx wrangler secret put TURNSTILE_SECRET       # Cloudflare Turnstile site → Secret
npx wrangler secret put DELIVERY_SECRET        # HMAC for /api/download — `openssl rand -hex 32`
```

Set `PUBLIC_TURNSTILE_SITE_KEY` on the **Pages** project env vars (matching public token).

## 3. Register your Dodo products

For every (slug, tier) you want to charge for, create a product in Dodo:

```bash
curl -X POST $DODO_BASE/products \
  -H "Authorization: Bearer $DODO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CSV Cleaner — Use It",
    "price": { "price": 900, "currency": "USD" }
  }'
# → { "id": "prod_abc123", ... }
```

Set the product ID as a Worker secret (one per tier):

```bash
npx wrangler secret put DODO_PRODUCT_csv_cleaner_mac_use   # paste prod_abc123
npx wrangler secret put DODO_PRODUCT_csv_cleaner_mac_own
# … repeat for each tier you want enabled today
```

Tiers without a product ID are not broken — the BE captures the buyer's intent
into D1 and returns `503 intent_captured` so the frontend shows
"Captured. We'll email you within 24 hours." Phase 1 = curated launch; you
do not need to wire all 40 tier products upfront.

### Optional: wire R2 auto-delivery per tier

When a tier has both a Dodo product AND a deliverable in R2, the buyer
gets an instant signed download link in the payment-success email instead
of "manual within 24h".

```bash
# Upload the deliverable
npx wrangler r2 object put \
  verified-bases-delivery/csv-cleaner-mac/use/v1.dmg \
  --file=./dist/csv-cleaner-1.0.dmg

# Tell the Worker where to find it
npx wrangler secret put DODO_DELIVERY_csv_cleaner_mac_use
# → paste:  csv-cleaner-mac/use/v1.dmg
```

The signed URL uses `DELIVERY_SECRET` (set above) and includes the
session_id as the public identifier. Rotating `DELIVERY_SECRET` invalidates
all outstanding links — fine for Phase 1.

## 4. Deploy the API Worker

```bash
cd api
make deploy
# → wrangler deploys verified-bases-api at https://verified-bases-api.<your-account>.workers.dev
```

Add the webhook URL to Dodo dashboard:
```
https://verified-bases-api.<your-account>.workers.dev/api/webhook/dodo
```
Subscribe to: `payment.succeeded`, `payment.failed`, `refund.succeeded`,
`refund.created`.

## 5. Deploy the Pages site

```bash
cd web
npm ci
npm run build

# Option A — connect the GitHub repo via Cloudflare dashboard. Build:
#   - Build command:    npm run build
#   - Build output dir: dist
#   - Node version:     22 (from web/.nvmrc)
#   - Env vars:
#       PUBLIC_CF_BEACON       = <token from Cloudflare Web Analytics>
#       PUBLIC_SITE_ORIGIN     = https://bases.sarthakagrawal.dev
#
# Option B — direct upload from CLI (faster first time):
npx wrangler pages deploy dist --project-name=verified-bases-web
```

## 6. Wire the service binding (Pages → Worker)

Cloudflare dashboard → Pages → `verified-bases-web` → Settings →
Functions → Service bindings:

```
Variable name : API
Service       : verified-bases-api
Environment   : production
```

Now `/api/*` on the Pages site routes via `web/functions/api/[[path]].ts`
into the Go Worker on the same edge.

## 7. Point DNS

Cloudflare dashboard → DNS → Add CNAME:
```
bases     CNAME   verified-bases-web.pages.dev    proxied
```

Then in Pages → `verified-bases-web` → Custom domains → Add
`bases.sarthakagrawal.dev`.

## 8. Smoke test the live deploy

```bash
curl -s https://bases.sarthakagrawal.dev/api/health | jq .
# → {"ok":true,"service":"verified-bases-api","phase":"1-manual-curated-store"}

# Submit form
curl -s -X POST https://bases.sarthakagrawal.dev/api/submit \
  -H 'Content-Type: application/json' \
  -d '{"title":"Smoke","email":"you@example.com","repo_or_demo":"https://github.com/x/y"}'

# Intent
curl -s -X POST https://bases.sarthakagrawal.dev/api/intent \
  -H 'Content-Type: application/json' \
  -d '{"slug":"csv-cleaner-mac","tier":"own","email":"you@example.com"}'
```

Verify in Cloudflare dashboard → D1 → `verified-bases-db` → Data view
that the rows landed.

## Subsequent deploys

```bash
# API change
cd api && make deploy

# Web change
git push   # if Pages is connected to Git
# or
cd web && npm run build && npx wrangler pages deploy dist --project-name=verified-bases-web
```

## Rollback

```bash
# Worker
npx wrangler rollback verified-bases-api

# Pages: use the dashboard "Rollback to this deployment" button.
```

## Operational gotchas

- **Dodo `DODO_ENV` ↔ key mismatch**: `live` keys against `test.dodopayments.com` (or vice versa) will surface as 401s from `createDodoCheckout`. Check the admin inbox for the `[bases] dodo checkout failed` email — it has the upstream body.
- **D1 + the migrations table**: D1's CLI tracks applied migrations. Adding migration files (e.g. `0002_…sql`) auto-applies via `make migrate-remote`.
- **KV eventual consistency**: rate-limit counters can lag a few seconds — design limits to absorb spikes. Phase 2 should swap KV for Cloudflare's Rate Limiting binding or a Durable Object.
- **Stale verifications**: the `lastVerified` date on each Base is hand-maintained in `web/src/data/bases.ts`. A future cron job + a `verifications` table can automate this.
