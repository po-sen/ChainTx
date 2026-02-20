ALTER TABLE app.payment_requests
  ADD COLUMN IF NOT EXISTS reconcile_lease_owner text;

ALTER TABLE app.payment_requests
  ADD COLUMN IF NOT EXISTS reconcile_lease_until timestamptz;

CREATE INDEX IF NOT EXISTS idx_payment_requests_reconcile_claim
  ON app.payment_requests (status, reconcile_lease_until, created_at, id);
