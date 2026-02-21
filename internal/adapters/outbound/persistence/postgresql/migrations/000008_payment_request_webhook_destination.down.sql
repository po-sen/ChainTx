ALTER TABLE app.webhook_outbox_events
  DROP COLUMN IF EXISTS destination_url;

ALTER TABLE app.payment_requests
  DROP COLUMN IF EXISTS webhook_url;
