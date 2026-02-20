DROP INDEX IF EXISTS idx_payment_requests_status_expires_at;
DROP INDEX IF EXISTS idx_payment_requests_status_chain_network;

ALTER TABLE app.payment_requests
  DROP CONSTRAINT IF EXISTS payment_requests_status_allowed;

ALTER TABLE app.payment_requests
  ADD CONSTRAINT payment_requests_status_non_empty CHECK (status <> '');
