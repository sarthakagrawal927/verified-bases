# Verified Software Bases

> Skip the blank prompt. Start from verified software.

A curated marketplace where buyers can preview, use, buy, own, remix, and launch
verified software **Bases** built by creators.

Bet: even when code generation is cheap, people still pay for **judgment,
verification, packaging, ownership, and a path to launch**.

## Layout

```
verified-bases/
├── docs/
│   ├── PRD-full.md           — 35-section product spec
│   └── PRD-distilled.md      — short version
├── web/                      — Astro 5 + React 19 + Tailwind v4 (Cloudflare Pages)
│   ├── src/
│   │   ├── data/bases.ts            — catalogue (empty; copy-paste schema inside)
│   │   ├── pages/                   — /, /bases, /bases/[slug], /trust,
│   │   │                              /collab, /about, /privacy, /terms, /refund, /404
│   │   ├── components/astro/        — Nav, Footer, Head, BaseCard, TierCard, TierModal
│   │   ├── layouts/BaseLayout.astro
│   │   └── styles/global.css        — design tokens + primitives
│   ├── functions/api/[[path]].ts    — Pages Function proxy → Worker
│   ├── scripts/icons.mjs            — Favicon + OG raster generation
│   ├── astro.config.mjs · package.json · tsconfig.json · wrangler.jsonc
│   └── .env.example
├── api/                      — Go on Cloudflare Workers (TinyGo + syumai/workers)
│   ├── main.go                      — router
│   ├── handlers/                    — health, submit, intent, checkout, webhook,
│   │                                  Dodo client + Standard Webhooks verify,
│   │                                  Resend client, D1 store, KV rate-limit
│   ├── migrations/0001_init.sql     — D1 schema
│   ├── wrangler.jsonc · Makefile · go.mod · .dev.vars.example
│   └── README.md                    — TinyGo / wrangler specifics
├── .github/workflows/ci.yml  — type-check + build web; vet + TinyGo build api
├── DEPLOY.md                 — first-time deploy walkthrough
├── PROJECT_STATUS.md
└── README.md (this file)
```

## Stack

Follows fleet standards (matches `fleet/sarthakagrawal`):

**Frontend** — Astro 5 static · React 19 islands · Tailwind v4 via the Vite
plugin · Lightning CSS transformer + minifier · Geist + Geist Mono variable
fonts · Cloudflare Pages.

**Backend** — Go 1.22 compiled with TinyGo to WASM · `syumai/workers` for
the fetch ⇄ net/http bridge · Cloudflare D1 (managed SQLite) for orders,
intents, submissions, webhook log · Cloudflare KV for rate-limit counters.

**Payments** — Dodo Payments (REST). Standard Webhooks signing.

**Email** — Resend (transactional).

## Design language

Dark engineered aesthetic with an **amber accent (`#f0b54a`)** — a
"verified stamp" / curator's seal feel. Retune the entire site by editing
`--color-accent` in `src/styles/global.css`.

## Run locally

```bash
# Frontend (no backend dep — modal flows degrade gracefully without /api).
cd web && npm install && npm run dev
# → http://localhost:4321

# Backend (separate terminal). Requires TinyGo.
cd api
brew install tinygo                     # one time
cp .dev.vars.example .dev.vars && $EDITOR .dev.vars
make migrate-local
make dev
# → http://localhost:8787
```

## Deploy

See `DEPLOY.md` for the full first-time walkthrough.

```bash
cd api && make deploy
cd web && npm run build && npx wrangler pages deploy dist --project-name=verified-bases-web
```

## Status

**Phase 1 — Manual Curated Store**, per `docs/PRD-full.md` §31. Buyers can
preview, place intent, or pay via Dodo Checkout; fulfilment (sending source,
running a remix, deploying for Launch Help) is manual until paid demand is
proven.

See `PROJECT_STATUS.md` for the open punch list.

## License

Buyer-ownership model. See each Base listing for its license terms; see
`/terms` and `/refund` on the live site for marketplace-level terms.
