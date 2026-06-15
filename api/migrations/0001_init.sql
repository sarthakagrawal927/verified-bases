-- verified-bases — initial D1 schema.
--
-- Apply: wrangler d1 execute verified-bases-db --remote --file=migrations/0001_init.sql
-- For local dev:  wrangler d1 execute verified-bases-db --local  --file=migrations/0001_init.sql

PRAGMA foreign_keys = ON;

------------------------------------------------------------
-- Creator submissions (queue for manual review).
------------------------------------------------------------
CREATE TABLE IF NOT EXISTS submissions (
  id            TEXT PRIMARY KEY,
  created_at    INTEGER NOT NULL,        -- unix seconds
  email         TEXT NOT NULL,
  name          TEXT,
  handle        TEXT,
  title         TEXT NOT NULL,
  one_liner     TEXT,
  category      TEXT,
  price         INTEGER,                 -- suggested USD
  repo_or_demo  TEXT NOT NULL,
  stack         TEXT,
  does_not_do   TEXT,
  limitations   TEXT,
  ai_gets_wrong TEXT,
  status        TEXT NOT NULL DEFAULT 'pending',  -- pending|needs_changes|verified|rejected
  reviewer_note TEXT
);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status, created_at DESC);

------------------------------------------------------------
-- Buyer intents (lead capture, regardless of payment outcome).
------------------------------------------------------------
CREATE TABLE IF NOT EXISTS intents (
  id          TEXT PRIMARY KEY,
  created_at  INTEGER NOT NULL,
  slug        TEXT NOT NULL,
  tier        TEXT NOT NULL,
  email       TEXT NOT NULL,
  note        TEXT,
  source      TEXT NOT NULL DEFAULT 'checkout',  -- checkout|preview|contact
  status      TEXT NOT NULL DEFAULT 'new'        -- new|contacted|won|lost
);
CREATE INDEX IF NOT EXISTS idx_intents_slug      ON intents(slug, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_intents_email     ON intents(email, created_at DESC);

------------------------------------------------------------
-- Paid orders (one row per Dodo checkout session, regardless of outcome).
------------------------------------------------------------
CREATE TABLE IF NOT EXISTS orders (
  dodo_session_id TEXT PRIMARY KEY,
  dodo_payment_id TEXT,
  created_at      INTEGER NOT NULL,
  paid_at         INTEGER,
  refunded_at     INTEGER,
  slug            TEXT NOT NULL,
  tier            TEXT NOT NULL,
  email           TEXT NOT NULL,
  amount_cents    INTEGER NOT NULL,
  currency        TEXT NOT NULL DEFAULT 'USD',
  status          TEXT NOT NULL DEFAULT 'pending', -- pending|paid|failed|refunded|disputed
  intent_id       TEXT REFERENCES intents(id)
);
CREATE INDEX IF NOT EXISTS idx_orders_email  ON orders(email, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_slug   ON orders(slug, created_at DESC);

------------------------------------------------------------
-- Webhook event log (idempotency + audit).
------------------------------------------------------------
CREATE TABLE IF NOT EXISTS webhook_events (
  webhook_id  TEXT PRIMARY KEY,         -- from `webhook-id` header
  received_at INTEGER NOT NULL,
  event_type  TEXT NOT NULL,
  payload     TEXT NOT NULL             -- raw body for replay/debug
);
