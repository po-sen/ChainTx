CREATE TABLE IF NOT EXISTS app.wallet_account_sync_events (
  id bigserial PRIMARY KEY,
  chain text NOT NULL,
  network text NOT NULL,
  keyset_id text NOT NULL,
  wallet_account_id text NOT NULL REFERENCES app.wallet_accounts (id),
  action text NOT NULL,
  match_source text NOT NULL,
  key_material_hash text NOT NULL,
  key_material_hash_algo text NOT NULL,
  details jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT wallet_account_sync_events_action_allowed CHECK (
    action IN ('reused', 'reactivated', 'rotated')
  ),
  CONSTRAINT wallet_account_sync_events_match_source_allowed CHECK (
    match_source IN ('active', 'legacy', 'unhashed')
  ),
  CONSTRAINT wallet_account_sync_events_hash_algo_allowed CHECK (
    key_material_hash_algo = 'hmac-sha256'
  ),
  CONSTRAINT wallet_account_sync_events_hash_hex CHECK (
    key_material_hash ~ '^[0-9a-f]{64}$'
  )
);

CREATE INDEX IF NOT EXISTS idx_wallet_account_sync_events_target_created
  ON app.wallet_account_sync_events (chain, network, keyset_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_wallet_account_sync_events_wallet_created
  ON app.wallet_account_sync_events (wallet_account_id, created_at DESC);
