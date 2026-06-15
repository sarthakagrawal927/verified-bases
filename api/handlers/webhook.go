package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DodoEnvelope is the Standard-Webhooks-style outer event we receive.
type DodoEnvelope struct {
	BusinessID string          `json:"business_id"`
	Type       string          `json:"type"`
	Timestamp  string          `json:"timestamp"`
	Data       json.RawMessage `json:"data"`
}

// DodoPaymentData is the shape we read out of `data` for payment.* events.
type DodoPaymentData struct {
	PayloadType string            `json:"payload_type"`
	PaymentID   string            `json:"payment_id"`
	SessionID   string            `json:"checkout_session_id"` // Dodo emits the originating session
	Amount      int               `json:"amount"`
	Currency    string            `json:"currency"`
	Email       string            `json:"customer_email"`
	Metadata    map[string]string `json:"metadata"`
}

// DodoWebhook receives Dodo Payments events. The verification path mirrors
// Standard Webhooks; on success the order row flips to paid / refunded and
// we email the buyer + admin.
func DodoWebhook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, `{"ok":false,"error":"read_body_failed"}`)
		return
	}

	if err := verifyDodoSignature(r.Header, body); err != nil {
		writeJSON(w, http.StatusUnauthorized, fmt.Sprintf(`{"ok":false,"error":"%s"}`, err.Error()))
		return
	}

	var env DodoEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		writeJSON(w, http.StatusBadRequest, `{"ok":false,"error":"invalid_envelope"}`)
		return
	}

	webhookID := r.Header.Get("webhook-id")
	ctx := r.Context()
	st, err := openStore()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, `{"ok":false,"error":"store_unavailable"}`)
		return
	}

	// Record-or-skip. Dodo retries on non-2xx — return 200 on duplicates so
	// they stop retrying, but only fan out side effects once.
	fresh, err := st.recordWebhook(ctx, webhookID, env.Type, body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, `{"ok":false,"error":"webhook_log_failed"}`)
		return
	}
	if !fresh {
		writeJSON(w, http.StatusOK, `{"ok":true,"status":"duplicate"}`)
		return
	}

	switch env.Type {
	case "payment.succeeded":
		handlePaymentSucceeded(ctx, st, env)
	case "payment.failed":
		handlePaymentFailed(env)
	case "refund.succeeded", "refund.created":
		handleRefund(ctx, st, env)
	default:
		// Unknown event — already logged. 200 to stop retries.
	}

	writeJSON(w, http.StatusOK, `{"ok":true}`)
}

func handlePaymentSucceeded(ctx context.Context, st *store, e DodoEnvelope) {
	var data DodoPaymentData
	_ = json.Unmarshal(e.Data, &data)

	if data.SessionID != "" {
		_ = st.markOrderPaid(ctx, data.SessionID, data.PaymentID)
	}

	slug := safeMetadata(data.Metadata, "slug")
	tier := safeMetadata(data.Metadata, "tier")

	// If a deliverable is configured in R2 for (slug, tier) and we can
	// sign a URL, include it in the buyer email — instant delivery.
	// Otherwise the buyer gets the manual-fulfilment note.
	var buyerBody string
	origin := env("PUBLIC_SITE_ORIGIN")
	if origin == "" {
		origin = "https://bases.sarthakagrawal.dev"
	}
	if downloadURL := autoDeliveryURL(origin, slug, tier, data.SessionID); downloadURL != "" {
		buyerBody = fmt.Sprintf(
			"Thanks for buying %s (%s tier).\n\nDownload it here (link expires after first successful download attempt, but is good for 7 days):\n\n%s\n\nReply to this email if anything's off.\n\n— Sarthak",
			slug, tier, downloadURL,
		)
	} else {
		buyerBody = fmt.Sprintf(
			"Thanks for buying %s (%s tier).\n\nI send the deliverable from sarthakagrawal927@gmail.com within 24 hours. If you don't see it by then, reply to this email.\n\n— Sarthak",
			slug, tier,
		)
	}

	if data.Email != "" {
		_, _ = sendEmail(resendEmail{
			To:      []string{data.Email},
			Subject: "Thanks — your Base is on the way",
			Text:    buyerBody,
		})
	}
	_, _ = sendEmail(resendEmail{
		To:      []string{adminInbox()},
		Subject: fmt.Sprintf("[bases] PAID: %s · %s", slug, tier),
		Text: fmt.Sprintf(
			"Payment succeeded.\n\nBase: %s\nTier: %s\nAmount: %.2f %s\nBuyer: %s\nDodo payment_id: %s\nDodo session_id: %s\nDelivery: %s\n",
			slug, tier, float64(data.Amount)/100, data.Currency, data.Email, data.PaymentID, data.SessionID,
			deliveryStatus(slug, tier),
		),
	})
}

// autoDeliveryURL returns a signed /api/download URL for the buyer if both
// the per-tier R2 key and the DELIVERY_SECRET are configured. Empty when
// either is missing — caller should fall back to manual fulfilment copy.
func autoDeliveryURL(origin, slug, tier, sessionID string) string {
	if deliveryKey(slug, tier) == "" {
		return ""
	}
	return buildDownloadURL(origin, sessionID)
}

func deliveryStatus(slug, tier string) string {
	if deliveryKey(slug, tier) != "" {
		return "auto (signed URL emailed)"
	}
	return "manual (action required — send the asset)"
}

func handlePaymentFailed(e DodoEnvelope) {
	var data DodoPaymentData
	_ = json.Unmarshal(e.Data, &data)
	_, _ = sendEmail(resendEmail{
		To:      []string{adminInbox()},
		Subject: "[bases] payment failed",
		Text: fmt.Sprintf(
			"Payment failed.\n\nBuyer: %s\nDodo session_id: %s\nMetadata: %+v\n",
			data.Email, data.SessionID, data.Metadata,
		),
	})
}

func handleRefund(ctx context.Context, st *store, e DodoEnvelope) {
	var data DodoPaymentData
	_ = json.Unmarshal(e.Data, &data)
	if data.SessionID != "" {
		_ = st.markOrderRefunded(ctx, data.SessionID)
	}
	_, _ = sendEmail(resendEmail{
		To:      []string{adminInbox()},
		Subject: "[bases] refund processed",
		Text: fmt.Sprintf(
			"Refund processed.\n\nDodo session_id: %s\nBuyer: %s\nMetadata: %+v\n",
			data.SessionID, data.Email, data.Metadata,
		),
	})
}

func safeMetadata(m map[string]string, k string) string {
	if m == nil {
		return ""
	}
	return m[k]
}
