package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	// Register the "d1" database/sql driver.
	_ "github.com/syumai/workers/cloudflare/d1"
)

// store provides typed access to the D1 binding via database/sql.
type store struct{ db *sql.DB }

func openStore() (*store, error) {
	// The string passed to sql.Open is the wrangler binding name ("DB" per
	// our wrangler.jsonc).
	db, err := sql.Open("d1", "DB")
	if err != nil {
		return nil, err
	}
	return &store{db: db}, nil
}

func now() int64 { return time.Now().Unix() }

func newID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// ── submissions ──────────────────────────────────────────────────────

func (s *store) insertSubmission(ctx context.Context, sub Submission) (string, error) {
	id := newID()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO submissions
			(id, created_at, email, name, handle, title, one_liner, category, price,
			 repo_or_demo, stack, does_not_do, limitations, ai_gets_wrong, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')
	`, id, now(), sub.Email, sub.Name, sub.Handle, sub.Title,
		sub.OneLiner, sub.Category, sub.Price, sub.RepoOrDemo, sub.Stack,
		sub.DoesNotDo, sub.Limitations, sub.AIGetsWrong)
	if err != nil {
		return "", err
	}
	return id, nil
}

// ── intents ──────────────────────────────────────────────────────────

func (s *store) insertIntent(ctx context.Context, slug, tier, email, note, source string) (string, error) {
	id := newID()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO intents (id, created_at, slug, tier, email, note, source, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'new')
	`, id, now(), slug, tier, email, note, source)
	if err != nil {
		return "", err
	}
	return id, nil
}

// ── orders ───────────────────────────────────────────────────────────

func (s *store) insertOrder(
	ctx context.Context, sessionID, slug, tier, email string, amountCents int, intentID string,
) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO orders
			(dodo_session_id, created_at, slug, tier, email, amount_cents, currency, status, intent_id)
		VALUES (?, ?, ?, ?, ?, ?, 'USD', 'pending', ?)
		ON CONFLICT(dodo_session_id) DO NOTHING
	`, sessionID, now(), slug, tier, email, amountCents, intentID)
	return err
}

func (s *store) markOrderPaid(ctx context.Context, sessionID, paymentID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE orders SET status = 'paid', paid_at = ?, dodo_payment_id = ?
		WHERE dodo_session_id = ?
	`, now(), paymentID, sessionID)
	return err
}

func (s *store) markOrderRefunded(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE orders SET status = 'refunded', refunded_at = ? WHERE dodo_session_id = ?
	`, now(), sessionID)
	return err
}

// lookupOrderBySession returns the slug, tier, buyer email, amount, and an
// existence flag for the order. Used by the /api/download handler to enforce
// "only the right buyer (via signed URL) for the right Base".
func (s *store) lookupOrderBySession(
	ctx context.Context, sessionID string,
) (slug, tier, email string, amountCents int, ok bool, err error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT slug, tier, email, amount_cents FROM orders
		WHERE dodo_session_id = ? AND status = 'paid'
		LIMIT 1
	`, sessionID)
	err = row.Scan(&slug, &tier, &email, &amountCents)
	if err == sql.ErrNoRows {
		return "", "", "", 0, false, nil
	}
	if err != nil {
		return "", "", "", 0, false, err
	}
	return slug, tier, email, amountCents, true, nil
}

// ── webhook idempotency ─────────────────────────────────────────────

// recordWebhook returns (fresh=true, nil) on first insert, (fresh=false, nil)
// on duplicate (already processed → caller should skip side effects).
func (s *store) recordWebhook(ctx context.Context, webhookID, eventType string, raw []byte) (bool, error) {
	if webhookID == "" {
		return false, errors.New("empty_webhook_id")
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_events (webhook_id, received_at, event_type, payload)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(webhook_id) DO NOTHING
	`, webhookID, now(), eventType, string(raw))
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}
