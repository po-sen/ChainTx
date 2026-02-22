DROP INDEX IF EXISTS idx_payment_request_settlements_request_state;
DROP TABLE IF EXISTS app.payment_request_settlements;

ALTER TABLE app.payment_requests
  DROP CONSTRAINT IF EXISTS payment_requests_status_allowed;

ALTER TABLE app.payment_requests
  ADD CONSTRAINT payment_requests_status_allowed
  CHECK (status IN ('pending', 'detected', 'confirmed', 'expired', 'failed'));
