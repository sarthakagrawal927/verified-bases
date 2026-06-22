package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// Resend — tiny REST client. We send transactional confirmations to the
// buyer and admin notifications to the operator inbox.

// Shared HTTP client — reuse across requests instead of allocating per call.
var httpClient = &http.Client{Timeout: 7 * time.Second}

type resendEmail struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
	ReplyTo string   `json:"reply_to,omitempty"`
}

// sendEmail posts to the Resend API. Returns the HTTP status code so the
// caller can decide whether to retry. Never panics — production code shouldn't
// blow up the request just because a notification failed.
func sendEmail(e resendEmail) (int, error) {
	apiKey := env("RESEND_API_KEY")
	if apiKey == "" {
		return http.StatusServiceUnavailable, nil
	}
	if e.From == "" {
		e.From = env("RESEND_FROM")
	}
	if e.From == "" {
		e.From = "Verified Bases <hello@bases.sarthakagrawal.dev>"
	}

	body, _ := json.Marshal(e)
	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func adminInbox() string {
	if s := env("ADMIN_INBOX_EMAIL"); s != "" {
		return s
	}
	return "sarthakagrawal927@gmail.com"
}
