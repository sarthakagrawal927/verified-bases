package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type CheckoutReq struct {
	Slug  string `json:"slug"`
	Tier  string `json:"tier"`
	Email string `json:"email"`
	Note  string `json:"note"`
}

// Checkout creates a Dodo hosted checkout session for the given (slug, tier).
//
// Three branches:
//  1. Unknown slug+tier         → 422 invalid_combo
//  2. Tier not wired to a Dodo product (no DODO_PRODUCT_… env) → 503 plus
//     we persist the intent so demand still surfaces.
//  3. Happy path → write a pending order row + return the Dodo checkout_url.
func Checkout(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var in CheckoutReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, `{"ok":false,"error":"invalid_json"}`)
		return
	}
	if clean(in.Slug) == "" || clean(in.Tier) == "" || clean(in.Email) == "" {
		writeJSON(w, http.StatusUnprocessableEntity, `{"ok":false,"error":"missing_required"}`)
		return
	}
	if !validTier(in.Tier) {
		writeJSON(w, http.StatusUnprocessableEntity, `{"ok":false,"error":"invalid_tier"}`)
		return
	}
	if in.Tier == "preview" {
		// /preview is free — the frontend opens the URL directly. If we got
		// here, treat as soft-error.
		writeJSON(w, http.StatusUnprocessableEntity, `{"ok":false,"error":"preview_is_free"}`)
		return
	}

	p, ok := lookupPrice(in.Slug, in.Tier)
	if !ok {
		writeJSON(w, http.StatusUnprocessableEntity, `{"ok":false,"error":"invalid_combo"}`)
		return
	}

	ctx := r.Context()
	st, err := openStore()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, `{"ok":false,"error":"store_unavailable"}`)
		return
	}

	// Capture the intent BEFORE talking to Dodo, so demand is preserved
	// even if the upstream call fails or the product isn't wired yet.
	intentID, err := st.insertIntent(ctx, in.Slug, in.Tier, in.Email, in.Note, "checkout")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, `{"ok":false,"error":"db_write_failed"}`)
		return
	}

	// No product configured → 503 with the friendly message the modal renders.
	if p.productID == "" {
		_, _ = sendEmail(resendEmail{
			To:      []string{adminInbox()},
			Subject: fmt.Sprintf("[bases] intent (no product yet): %s · %s", in.Slug, in.Tier),
			Text: fmt.Sprintf(
				"Buyer wants %s — %s tier ($%.2f) but DODO_PRODUCT_%s_%s is unset.\n\nEmail: %s\nNote: %s\nIntent ID: %s\n",
				in.Slug, in.Tier, float64(p.amountCents)/100,
				envSafe(in.Slug), in.Tier, in.Email, in.Note, intentID,
			),
			ReplyTo: in.Email,
		})
		writeJSON(w, http.StatusServiceUnavailable, `{"ok":true,"status":"intent_captured","message":"Captured. We'll email you with a payment link within 24 hours."}`)
		return
	}

	site := siteOrigin(r)
	sess, code, err := createDodoCheckout(CheckoutSessionReq{
		ProductCart: []dodoCartItem{{ProductID: p.productID, Quantity: 1}},
		Customer:    dodoCustomer{Email: in.Email},
		ReturnURL:   site + "/bases/" + in.Slug + "?status=ok",
		CancelURL:   site + "/bases/" + in.Slug + "?status=cancelled",
		Metadata: map[string]string{
			"slug":      in.Slug,
			"tier":      in.Tier,
			"intent_id": intentID,
		},
	})
	if err != nil {
		// Surface to admin, generic to buyer.
		_, _ = sendEmail(resendEmail{
			To:      []string{adminInbox()},
			Subject: "[bases] dodo checkout failed",
			Text:    fmt.Sprintf("Intent %s for %s/%s failed: %v\n", intentID, in.Slug, in.Tier, err),
		})
		writeJSON(w, code, `{"ok":false,"error":"checkout_upstream_failed"}`)
		return
	}

	if err := st.insertOrder(ctx, sess.SessionID, in.Slug, in.Tier, in.Email, p.amountCents, intentID); err != nil {
		// Not fatal — webhook will reconcile by metadata anyway.
		_, _ = sendEmail(resendEmail{
			To:      []string{adminInbox()},
			Subject: "[bases] orders insert failed (reconcile via webhook)",
			Text:    fmt.Sprintf("session=%s intent=%s err=%v\n", sess.SessionID, intentID, err),
		})
	}

	writeJSON(w, http.StatusOK, fmt.Sprintf(
		`{"ok":true,"session_id":"%s","checkout_url":"%s"}`,
		sess.SessionID, sess.CheckoutURL,
	))
}

// siteOrigin returns the canonical site origin for return/cancel URLs.
// Prefers PUBLIC_SITE_ORIGIN env, falls back to the inbound Origin header,
// then to the canonical prod URL.
func siteOrigin(r *http.Request) string {
	if s := env("PUBLIC_SITE_ORIGIN"); s != "" {
		return s
	}
	if o := r.Header.Get("Origin"); o != "" {
		return o
	}
	return "https://bases.sarthakagrawal.dev"
}
