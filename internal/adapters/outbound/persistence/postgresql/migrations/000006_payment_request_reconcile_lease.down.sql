DROP INDEX IF EXISTS idx_payment_requests_reconcile_claim;

ALTER TABLE app.payment_requests
  DROP COLUMN IF EXISTS reconcile_lease_until;

ALTER TABLE app.payment_requests
  DROP COLUMN IF EXISTS reconcile_lease_owner;
