package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
)

// Cloudflare Turnstile — verifies a token issued by the client widget.
//
// We pass the Worker secret + the user's token + their CF-Connecting-IP to
// `siteverify`. The endpoint replies with {"success": true, ...} on a pass.
//
// In tests / local dev we accept the well-known Cloudflare test token,
// keyed by setting TURNSTILE_SECRET="1x0000000000000000000000000000000AA"
// (always-passes test secret).

const turnstileSiteVerify = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type turnstileResp struct {
	Success     bool     `json:"success"`
	ErrorCodes  []string `json:"error-codes"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
}

// verifyTurnstile returns nil when the token is valid. When the secret env
// var is unset we fail OPEN so local dev / preview environments without
// the widget can still submit — log it visibly in the response header.
func verifyTurnstile(token, ip string) (ok bool, errorCode string) {
	secret := env("TURNSTILE_SECRET")
	if secret == "" {
		// No secret configured — treat as "not enforced".
		return true, "not_enforced"
	}
	if clean(token) == "" {
		return false, "missing_token"
	}

	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if ip != "" {
		form.Set("remoteip", ip)
	}

	req, err := http.NewRequest(http.MethodPost, turnstileSiteVerify, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return false, "request_build_failed"
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, "siteverify_unreachable"
	}
	defer resp.Body.Close()

	var out turnstileResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, "siteverify_bad_response"
	}
	if !out.Success {
		if len(out.ErrorCodes) > 0 {
			return false, out.ErrorCodes[0]
		}
		return false, "verification_failed"
	}
	return true, ""
}
