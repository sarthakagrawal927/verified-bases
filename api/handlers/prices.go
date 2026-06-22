package handlers

// priceTable mirrors `web/src/data/bases.ts` for the slug+tier combinations
// that have a real Dodo Payments product. It MUST be kept in sync when:
//   - a new Base is added to the catalogue,
//   - a tier's price changes,
//   - a Dodo product is (re-)registered.
//
// Empty by default. Once you ship your first Base, add its tiers below.
// The "preview" tier never has a row — it's always free, no checkout.
//
// Example entries (delete after pasting your real ones):
//
//   "your-base-slug:use":   {amountCents:  1900, displayName: "Your Base — Use It"},
//   "your-base-slug:own":   {amountCents:  7900, displayName: "Your Base — Own It"},
//   "your-base-slug:remix": {amountCents:  4900, displayName: "Your Base — Remix Run"},
//
// Then create the corresponding Dodo Payments product:
//
//   curl -X POST https://test.dodopayments.com/products \
//        -H "Authorization: Bearer $DODO_API_KEY" \
//        -H "Content-Type: application/json" \
//        -d '{"name":"Your Base — Own It","price":{"price":7900,"currency":"USD"}}'
//
// and set its product ID as a Worker secret:
//
//   wrangler secret put DODO_PRODUCT_your_base_slug_own

type price struct {
	productID   string // Dodo Payments product ID (set per environment via env override below)
	amountCents int    // USD cents — authoritative price the BE charges
	displayName string // shown in admin emails / order rows
}

// priceTable maps "<slug>:<tier>" → price. First Base: TinyGPT Specialist Starter.
var priceTable = map[string]price{
	"tinygpt-specialist-starter:use": {
		amountCents: 2900,
		displayName: "TinyGPT Specialist Starter — Use It",
	},
	"tinygpt-specialist-starter:own": {
		amountCents: 9900,
		displayName: "TinyGPT Specialist Starter — Own It",
	},
	"tinygpt-specialist-starter:remix": {
		amountCents: 7900,
		displayName: "TinyGPT Specialist Starter — Remix Run",
	},
	"tinygpt-specialist-starter:launch": {
		amountCents: 19900,
		displayName: "TinyGPT Specialist Starter — Launch Help",
	},
}

// lookupPrice returns the price entry plus the per-environment Dodo product
// ID. Product IDs come from env vars rather than hardcoded so we can keep
// separate test/live IDs without code changes:
//
//	DODO_PRODUCT_tinygpt_specialist_starter_own = prod_abc123
//	DODO_PRODUCT_tinygpt_specialist_starter_use = prod_def456
//	DODO_PRODUCT_tinygpt_specialist_starter_remix = prod_ghi789
//	DODO_PRODUCT_tinygpt_specialist_starter_launch = prod_jkl012
//
// Underscores replace `-` and `:` in the slug:tier key.
func lookupPrice(slug, tier string) (price, bool) {
	key := slug + ":" + tier
	p, ok := priceTable[key]
	if !ok {
		return price{}, false
	}
	envKey := "DODO_PRODUCT_" + envSafe(slug) + "_" + tier
	p.productID = env(envKey)
	return p, true
}

func envSafe(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == ':' {
			out = append(out, '_')
		} else {
			out = append(out, c)
		}
	}
	return string(out)
}
