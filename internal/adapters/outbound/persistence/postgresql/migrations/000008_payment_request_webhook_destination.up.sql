ALTER TABLE app.payment_requests
  ADD COLUMN IF NOT EXISTS webhook_url text;

ALTER TABLE app.webhook_outbox_events
  ADD COLUMN IF NOT EXISTS destination_url text;

UPDATE app.webhook_outbox_events AS e
SET destination_url = pr.webhook_url
FROM app.payment_requests AS pr
WHERE e.payment_request_id = pr.id
  AND (e.destination_url IS NULL OR btrim(e.destination_url) = '')
  AND pr.webhook_url IS NOT NULL
  AND btrim(pr.webhook_url) <> '';

UPDATE app.webhook_outbox_events
SET
  delivery_status = 'failed',
  last_error = 'webhook destination_url missing',
  lease_owner = NULL,
  lease_until = NULL,
  updated_at = now()
WHERE delivery_status = 'pending'
  AND (destination_url IS NULL OR btrim(destination_url) = '');
