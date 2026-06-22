-- verified-bases — add missing index on orders.intent_id FK.
--
-- Apply: wrangler d1 execute verified-bases-db --remote --file=migrations/0002_intent_index.sql
-- For local dev:  wrangler d1 execute verified-bases-db --local  --file=migrations/0002_intent_index.sql

CREATE INDEX IF NOT EXISTS idx_orders_intent ON orders(intent_id);
