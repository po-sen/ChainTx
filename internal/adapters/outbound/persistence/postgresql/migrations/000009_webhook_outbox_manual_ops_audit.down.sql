ALTER TABLE app.webhook_outbox_events
  DROP CONSTRAINT IF EXISTS webhook_outbox_manual_last_action_allowed;

ALTER TABLE app.webhook_outbox_events
  DROP COLUMN IF EXISTS manual_last_at,
  DROP COLUMN IF EXISTS manual_last_actor,
  DROP COLUMN IF EXISTS manual_last_action;
