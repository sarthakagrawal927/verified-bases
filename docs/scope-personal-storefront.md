# Scope: personal-tools storefront + integrable product pages

**Decision (2026-06-22):** verified-bases is a **storefront for my own software**,
not a marketplace. Validate whether people pay for the tools I've already built.

---

## Scope — locked
- **Personal only.** Drop the marketplace / creator-collab / remix / multi-seller
  ambition and `/collab`. Every Base is Sarthak-built.
- **Launch catalogue = downloadable/ownable apps** (the right kind for "buy · own
  · launch"):
  - **pace** — macOS voice companion
  - **CodeVetter** — desktop AI code review (already on GitHub Releases)
  - **tinygpt** — local LLM factory/runtime
  - Web SaaS (linkchat, reader, …) are *not* Bases — you don't "own" a hosted app.

## What already exists (extend, don't rebuild)
- `web/src/data/bases.ts` — catalogue data
- `web/src/pages/bases/index.astro` — storefront listing
- `web/src/pages/bases/[slug].astro` — **per-product page** (dynamic route)

## New: per-product page that "integrates across" the fleet
**Goal:** one canonical product record per tool → rendered as the storefront
product page **and** embeddable across other surfaces: the tool's own site
(pace/website, CodeVetter site, tinygpt playground), the portfolio
(sarthakagrawal.dev), the fleet factory page, and GitHub READMEs.

### Decision needed — single source of truth for product metadata
- **A) keep `bases.ts` local** — simplest; "integrate across" means other surfaces import/copy it.
- **B) saas-maker project registry as SoT (recommended)** — projects already hold
  name/desc/git_url; extend with `price`, `downloadUrl`, `screenshots`,
  `platform`, `status`. verified-bases + portfolio + each tool fetch `/v1/projects`.
  True DRY, aligns with the tier-1 system-of-record we kept.

### Reusable render unit
- A shared **product card / page** component (data → HTML island) + an **SVG card**
  for READMEs.
- **Reuse placard.** `github.com/sarthakagrawal927/placard` already renders
  `project.json → embeddable SVG/PNG card` — that *is* the README-embed piece.
  Revive/fold it rather than rebuild.

## Validation-first (the only thing that gates everything)
- Working **checkout** for ≥1 product. Dodo is already wired; if it stalls on
  provisioning, a lighter option (Gumroad / Polar / Lemonsqueezy) validates demand
  in a day. **First sale before more infra.**

## Acceptance criteria
- [ ] Catalogue scoped to personal tools; marketplace/collab UI removed.
- [ ] One canonical product record per tool (source A or B decided).
- [ ] Per-product page renders from that record.
- [ ] Same record/component embedded on ≥1 external surface (a tool's own site or the portfolio).
- [ ] Working checkout for ≥1 product (pace / CodeVetter / tinygpt).

## Out of scope
Creator onboarding, remix UX, multi-seller, reviews/ratings, payouts.

## Open questions
- SoT: saas-maker registry vs per-repo `product.json` vs `bases.ts`?
- Embed format: HTML/JS island vs SVG card (placard) vs both?
- Checkout: finish Dodo (built) or use a hosted option to validate faster?
