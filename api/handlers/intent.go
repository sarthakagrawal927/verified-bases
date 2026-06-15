package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type IntentReq struct {
	Slug  string `json:"slug"`
	Tier  string `json:"tier"`
	Email string `json:"email"`
	Note  string `json:"note"`
}

// Intent captures a buyer's interest in a tier without going through Dodo
// (e.g. tiers that don't have a configured product yet, or the dedicated
// /api/intent endpoint that bypasses checkout entirely).
func Intent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var in IntentReq
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

	ctx := r.Context()
	st, err := openStore()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, `{"ok":false,"error":"store_unavailable"}`)
		return
	}
	id, err := st.insertIntent(ctx, in.Slug, in.Tier, in.Email, in.Note, "contact")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, `{"ok":false,"error":"db_write_failed"}`)
		return
	}

	_, _ = sendEmail(resendEmail{
		To:      []string{adminInbox()},
		Subject: fmt.Sprintf("[bases] buyer intent: %s · %s", in.Slug, in.Tier),
		Text: fmt.Sprintf(
			"Buyer intent (id %s)\n\nBase: %s\nTier: %s\nEmail: %s\nNote: %s\n",
			id, in.Slug, in.Tier, in.Email, in.Note,
		),
		ReplyTo: in.Email,
	})

	writeJSON(w, http.StatusAccepted,
		fmt.Sprintf(`{"ok":true,"id":"%s","status":"received","next":"we_email_within_24h"}`, id),
	)
}

func validTier(t string) bool {
	switch t {
	case "preview", "use", "own", "remix", "launch", "hosting":
		return true
	}
	return false
}
