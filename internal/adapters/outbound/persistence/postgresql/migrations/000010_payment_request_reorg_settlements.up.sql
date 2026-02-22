ALTER TABLE app.payment_requests
  DROP CONSTRAINT IF EXISTS payment_requests_status_allowed;

ALTER TABLE app.payment_requests
  ADD CONSTRAINT payment_requests_status_allowed
  CHECK (status IN ('pending', 'detected', 'confirmed', 'reorged', 'expired', 'failed'));

CREATE TABLE IF NOT EXISTS app.payment_request_settlements (
  payment_request_id text NOT NULL REFERENCES app.payment_requests (id) ON DELETE CASCADE,
  evidence_ref text NOT NULL,
  amount_minor numeric(78,0) NOT NULL,
  confirmations integer NOT NULL,
  block_height bigint,
  block_hash text,
  is_canonical boolean NOT NULL DEFAULT TRUE,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  first_seen_at timestamptz NOT NULL,
  last_seen_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (payment_request_id, evidence_ref),
  CONSTRAINT payment_request_settlements_amount_non_negative CHECK (amount_minor >= 0),
  CONSTRAINT payment_request_settlements_confirmations_non_negative CHECK (confirmations >= 0),
  CONSTRAINT payment_request_settlements_metadata_size CHECK (octet_length(metadata::text) <= 4096)
);

CREATE INDEX IF NOT EXISTS idx_payment_request_settlements_request_state
  ON app.payment_request_settlements (payment_request_id, is_canonical, confirmations);
