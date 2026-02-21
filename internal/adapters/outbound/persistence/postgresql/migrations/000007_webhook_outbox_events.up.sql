CREATE TABLE IF NOT EXISTS app.webhook_outbox_events (
  id bigserial PRIMARY KEY,
  event_id text NOT NULL UNIQUE,
  event_type text NOT NULL,
  payment_request_id text NOT NULL REFERENCES app.payment_requests (id) ON DELETE CASCADE,
  payload jsonb NOT NULL,
  delivery_status text NOT NULL DEFAULT 'pending',
  attempts integer NOT NULL DEFAULT 0,
  max_attempts integer NOT NULL DEFAULT 8,
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  lease_owner text,
  lease_until timestamptz,
  last_error text,
  delivered_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT webhook_outbox_status_allowed
    CHECK (delivery_status IN ('pending', 'delivered', 'failed')),
  CONSTRAINT webhook_outbox_attempts_non_negative CHECK (attempts >= 0),
  CONSTRAINT webhook_outbox_max_attempts_positive CHECK (max_attempts > 0),
  CONSTRAINT webhook_outbox_payload_size CHECK (octet_length(payload::text) <= 16384)
);

CREATE INDEX IF NOT EXISTS idx_webhook_outbox_claim
  ON app.webhook_outbox_events (delivery_status, next_attempt_at, lease_until, created_at, id);

CREATE INDEX IF NOT EXISTS idx_webhook_outbox_payment_request
  ON app.webhook_outbox_events (payment_request_id, created_at DESC);
