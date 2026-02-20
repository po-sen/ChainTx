ALTER TABLE app.payment_requests
  DROP CONSTRAINT IF EXISTS payment_requests_status_non_empty;

ALTER TABLE app.payment_requests
  DROP CONSTRAINT IF EXISTS payment_requests_status_allowed;
ALTER TABLE app.payment_requests
  ADD CONSTRAINT payment_requests_status_allowed
  CHECK (status IN ('pending', 'detected', 'confirmed', 'expired', 'failed'));

CREATE INDEX IF NOT EXISTS idx_payment_requests_status_expires_at
  ON app.payment_requests (status, expires_at);

CREATE INDEX IF NOT EXISTS idx_payment_requests_status_chain_network
  ON app.payment_requests (status, chain, network);
