package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Dodo Payments — minimal client over the REST API.
//
// We do NOT use the Dodo Go SDK to keep the TinyGo binary small (the SDK
// pulls in reflect-heavy code paths). Standard Webhooks signature
// verification is implemented locally.

// dodoBaseURL returns the API root for the current environment.
func dodoBaseURL() string {
	if env("DODO_ENV") == "live" {
		return "https://live.dodopayments.com"
	}
	return "https://test.dodopayments.com"
}

// CheckoutSessionReq is the body we POST to Dodo /checkouts.
type CheckoutSessionReq struct {
	ProductCart []dodoCartItem    `json:"product_cart"`
	Customer    dodoCustomer      `json:"customer"`
	ReturnURL   string            `json:"return_url"`
	CancelURL   string            `json:"cancel_url"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type dodoCartItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type dodoCustomer struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// CheckoutSessionRes is the response shape Dodo returns from /checkouts.
type CheckoutSessionRes struct {
	SessionID   string `json:"session_id"`
	CheckoutURL string `json:"checkout_url"`
}

// createDodoCheckout posts to Dodo's /checkouts endpoint and returns the
// hosted checkout URL the buyer should be redirected to.
func createDodoCheckout(req CheckoutSessionReq) (*CheckoutSessionRes, int, error) {
	apiKey := env("DODO_API_KEY")
	if apiKey == "" {
		return nil, http.StatusServiceUnavailable, errors.New("dodo_api_key_unset")
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest(http.MethodPost, dodoBaseURL()+"/checkouts", bytes.NewReader(body))
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Bubble Dodo's body up to the admin log channel; don't leak to buyer.
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		return nil, http.StatusBadGateway, &dodoErr{status: resp.StatusCode, body: buf.String()}
	}

	var out CheckoutSessionRes
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, http.StatusBadGateway, err
	}
	return &out, http.StatusOK, nil
}

type dodoErr struct {
	status int
	body   string
}

func (e *dodoErr) Error() string {
	return "dodo " + strconv.Itoa(e.status) + ": " + e.body
}

// ─────────────────────────────────────────────────────────────────────
// Standard Webhooks signature verification
// ─────────────────────────────────────────────────────────────────────
//
// Per Dodo docs (and the Standard Webhooks spec):
//   - Signed payload: "<webhook-id>.<webhook-timestamp>.<raw-body>"
//   - Algorithm:      HMAC-SHA256 with the secret from the Dodo dashboard
//   - Header `webhook-signature` carries one or more "vN,<base64-sig>"
//     entries, space-separated. Any one matching value passes verification.
//   - Reject signatures older than 5 minutes (replay window).
//
// The secret string from the Dodo dashboard is the HMAC key as-is.

const webhookReplayWindow = 5 * 60 // seconds

func verifyDodoSignature(headers http.Header, body []byte) error {
	id := headers.Get("webhook-id")
	ts := headers.Get("webhook-timestamp")
	sig := headers.Get("webhook-signature")
	if id == "" || ts == "" || sig == "" {
		return errors.New("missing_webhook_headers")
	}

	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.New("bad_webhook_timestamp")
	}
	if delta := time.Now().Unix() - tsInt; delta < -webhookReplayWindow || delta > webhookReplayWindow {
		return errors.New("webhook_timestamp_outside_window")
	}

	secret := env("DODO_WEBHOOK_SECRET")
	if secret == "" {
		return errors.New("dodo_webhook_secret_unset")
	}

	signedPayload := id + "." + ts + "." + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	for _, candidate := range strings.Split(sig, " ") {
		// Header values look like "v1,base64sig" — split on the comma.
		comma := strings.IndexByte(candidate, ',')
		if comma <= 0 {
			continue
		}
		got := candidate[comma+1:]
		if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1 {
			return nil
		}
	}
	return errors.New("signature_mismatch")
}
