# PROJECT_STATUS — verified-bases

_Last updated: 2026-06-15_

## Phase

**Phase 1 — Personal storefront, soft creator door**.

Every Base on the site is currently built and sold by Sarthak. The site is
positioned as a personal verified-software shop with a `/collab` page that
captures inbound from other builders without committing to running a
marketplace UX. If `/collab` produces something good, that's the path back
to the original marketplace thesis from the PRD.

Goal: test paid demand on YOUR work first, without supply-side
chicken-and-egg. Checkout, intent capture, persistence, notifications, and
file delivery are automated; manual review for any incoming `/collab`
submissions.

## Shipped

### Docs
- `docs/PRD-full.md` — 35-section product spec
- `docs/PRD-distilled.md`
- `DEPLOY.md` — first-time deploy walkthrough
- `/privacy`, `/terms`, `/refund` legal pages

### Frontend (`web/`)
- Astro 5 / React 19 / Tailwind v4 / Lightning CSS / Geist fonts
- Design tokens with amber "verified stamp" accent
- Pages: `/`, `/bases`, `/bases/[slug]`, `/trust`, `/collab`, `/about`,
  `/privacy`, `/terms`, `/refund`, `/404`
- Bases catalogue empty by default — `src/data/bases.ts` ships the type
  + schema + a copy-paste example shape, ready for the first real Base
- Homepage, `/bases`, and Categories all handle the empty state cleanly
  ("first Base shipping soon" → "I'm building it now")
- TierModal — buyer enters email → POST `/api/checkout` → Dodo hosted
  checkout (or `503 intent_captured` fallback when the tier's product
  isn't wired)
- Pages Function proxy `web/functions/api/[[path]].ts` → service binding
- Mobile-responsive (verified at 390px viewport)
- OG image (1200×630 SVG) + favicon set + apple-touch-icon (rasterized
  via `npm run prebuild`)
- Cloudflare Web Analytics (cookieless, opt-in via `PUBLIC_CF_BEACON`)
- Verified clean: 18 pages built, 0 type errors

### Backend (`api/`, Go on Cloudflare Workers, full Go via wasm/js)
- Endpoints: `GET /api/health`, `POST /api/submit`, `POST /api/intent`,
  `POST /api/checkout`, `POST /api/webhook/dodo`, `GET /api/download`
- D1 schema in `migrations/0001_init.sql`: `submissions`, `intents`,
  `orders`, `webhook_events` (with idempotency)
- Dodo Payments REST client (test + live), Standard Webhooks signature
  verification (HMAC-SHA256, 5-min replay window)
- Resend transactional emails to buyer + admin on every meaningful event
- Cloudflare Turnstile verification on `/api/submit`
- Cloudflare R2 binding `DELIVERY` — `/api/download` streams signed buyer
  downloads (HMAC of session_id with `DELIVERY_SECRET`)
- Auto-delivery URL injected into payment-success email when both Dodo
  product + R2 delivery key are configured; falls back to manual otherwise
- KV-backed rate limit (20/min/IP/endpoint, with `webhook` + `download` exempt)
- Worker secrets contract documented in `wrangler.jsonc` + `.dev.vars.example`
- Pricing table `handlers/prices.go` mirrors tier prices in dollars/cents
- Cloudflare-native rollup: D1 + KV + R2 + Turnstile + Workers + Pages.
  Only non-CF deps: Dodo Payments (no CF equivalent) + Resend (CF's
  outbound email is restricted to verified recipients, not viable for
  buyer confirmations as of 2026).
- Build artefact: 11 MB raw, **2.9 MB gzipped** — fits Workers Free tier

### CI / Ops
- `.github/workflows/ci.yml` — web: type-check + build + artifact; api:
  go vet + TinyGo build + artifact
- `wrangler.jsonc` on both sides with placeholder IDs ready for
  `wrangler d1 create` / `wrangler kv:namespace create`

## Open (before first paid buyer)

- [ ] Add your first Base to `web/src/data/bases.ts` (paste the example shape, fill it in)
- [ ] Mirror the price rows into `api/handlers/prices.go`
- [ ] Provision Cloudflare resources (D1 db, KV namespace, R2 bucket, Turnstile site)
      and paste IDs into `api/wrangler.jsonc`
- [ ] Set Worker secrets: `DODO_API_KEY`, `DODO_WEBHOOK_SECRET`,
      `RESEND_API_KEY`, `TURNSTILE_SECRET`, `DELIVERY_SECRET`
- [ ] Register the first Base's tier(s) as Dodo products and set
      `DODO_PRODUCT_<slug>_<tier>` secrets
- [ ] (Optional) Upload deliverable file to R2 and set
      `DODO_DELIVERY_<slug>_<tier>` for instant auto-delivery
- [ ] Set `PUBLIC_TURNSTILE_SITE_KEY` on the Pages project env
- [ ] Deploy via `make deploy` + `wrangler pages deploy`, wire DNS to
      `bases.sarthakagrawal.dev`
- [ ] Register webhook URL in Dodo dashboard and subscribe to
      `payment.succeeded`, `payment.failed`, `refund.*`
- [ ] End-to-end smoke: `/collab` form (Turnstile + email lands) →
      checkout in test mode (D1 row appears) → refund a test order
      (email + D1 reflect it)

## Phase 2 (after paid demand validated)

- [ ] Automated GitHub invite on Own-It purchase
- [ ] R2 signed-URL delivery for Use-It tier
- [ ] Creator dashboard (sales + payout status)
- [ ] Automated last-verified checks + freshness alerts
- [ ] Swap KV rate-limit for Cloudflare Rate Limiting binding or a DO

## Validation metric

> Will people pay for verified outcomes instead of prompting from scratch?

Tracked through `orders` (paid revenue) and `intents` (demand before
checkout is wired). Both queryable from D1.

## Cross-fleet notes

- Mirrors fleet's Astro + Tailwind v4 + Lightning CSS pattern.
- Amber accent intentionally distinct from sibling sites (teal/lime).
- Free of cross-talk with `fleet/free-ai` (see memory).
