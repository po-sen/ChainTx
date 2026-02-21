ALTER TABLE app.webhook_outbox_events
  ADD COLUMN IF NOT EXISTS manual_last_action text,
  ADD COLUMN IF NOT EXISTS manual_last_actor text,
  ADD COLUMN IF NOT EXISTS manual_last_at timestamptz;

ALTER TABLE app.webhook_outbox_events
  DROP CONSTRAINT IF EXISTS webhook_outbox_manual_last_action_allowed;

ALTER TABLE app.webhook_outbox_events
  ADD CONSTRAINT webhook_outbox_manual_last_action_allowed
  CHECK (manual_last_action IS NULL OR manual_last_action IN ('requeue', 'cancel'));
