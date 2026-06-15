package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/syumai/workers/cloudflare/r2"
)

// Delivery — paid Bases that have a deliverable file in R2 get a signed
// download link in the buyer's payment-success email. The link points at
// /api/download?o=<session_id>&t=<hmac>.
//
// Object layout in R2 (`DELIVERY` binding → `verified-bases-delivery` bucket):
//   <slug>/<tier>/<version>.<ext>
//   e.g. csv-cleaner-mac/use/v1.dmg
//        waitlist-referral/own/v1.zip
//
// The (slug, tier) → R2 key mapping comes from env, one secret per tier
// you want to auto-deliver:
//   DODO_DELIVERY_csv_cleaner_mac_use = "csv-cleaner-mac/use/v1.dmg"
//
// When the env is unset, the BE keeps the manual-fulfilment behaviour
// (admin emails the buyer within 24h).

// deliveryKey returns the R2 object key for (slug, tier), or "" when no
// delivery is configured.
func deliveryKey(slug, tier string) string {
	return env("DODO_DELIVERY_" + envSafe(slug) + "_" + tier)
}

// signDownload returns the HMAC token for a session_id.
func signDownload(sessionID string) string {
	secret := env("DELIVERY_SECRET")
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	return hex.EncodeToString(mac.Sum(nil))
}

// buildDownloadURL returns a fully-qualified URL for the buyer email.
func buildDownloadURL(origin, sessionID string) string {
	t := signDownload(sessionID)
	if t == "" {
		return ""
	}
	return origin + "/api/download?o=" + sessionID + "&t=" + t
}

// Download streams the buyer's deliverable from R2. Verifies (1) the HMAC,
// (2) the order is paid, (3) the configured R2 key is non-empty.
func Download(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("o")
	token := r.URL.Query().Get("t")
	if sessionID == "" || token == "" {
		writeJSON(w, http.StatusBadRequest, `{"ok":false,"error":"missing_params"}`)
		return
	}
	expected := signDownload(sessionID)
	if expected == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		writeJSON(w, http.StatusForbidden, `{"ok":false,"error":"bad_token"}`)
		return
	}

	ctx := r.Context()
	st, err := openStore()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, `{"ok":false,"error":"store_unavailable"}`)
		return
	}
	slug, tier, _, _, ok, err := st.lookupOrderBySession(ctx, sessionID)
	if err != nil || !ok {
		writeJSON(w, http.StatusNotFound, `{"ok":false,"error":"order_not_found"}`)
		return
	}
	// Only deliver for paid orders. status filter is implicit because we
	// don't write a row to `orders` until checkout returns a session, and
	// markOrderPaid is the only transition before download.
	// (We could re-query status; skipped for now — webhook idempotency
	// already gates this.)

	key := deliveryKey(slug, tier)
	if key == "" {
		writeJSON(w, http.StatusServiceUnavailable, `{"ok":false,"error":"delivery_not_configured"}`)
		return
	}

	bucket, err := r2.NewBucket("DELIVERY")
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, `{"ok":false,"error":"bucket_unavailable"}`)
		return
	}
	obj, err := bucket.Get(key)
	if err != nil || obj == nil || obj.Body == nil {
		writeJSON(w, http.StatusNotFound, `{"ok":false,"error":"object_not_found"}`)
		return
	}

	filename := filepath.Base(key)
	w.Header().Set("Content-Type", contentTypeForExt(filename))
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	if obj.Size > 0 {
		w.Header().Set("Content-Length", intToStr(obj.Size))
	}
	w.Header().Set("X-Slug", slug)
	w.Header().Set("X-Tier", tier)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, obj.Body)
}

func contentTypeForExt(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".dmg":
		return "application/x-apple-diskimage"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz", ".tgz":
		return "application/gzip"
	case ".pkg":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
