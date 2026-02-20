ALTER TABLE app.wallet_accounts
  ADD COLUMN IF NOT EXISTS key_material_hash text,
  ADD COLUMN IF NOT EXISTS key_material_hash_algo text;

ALTER TABLE app.wallet_accounts
  DROP CONSTRAINT IF EXISTS wallet_accounts_chain_network_keyset_unique;

DROP INDEX IF EXISTS idx_wallet_accounts_chain_network_keyset_active;
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_accounts_chain_network_keyset_active
  ON app.wallet_accounts (chain, network, keyset_id)
  WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_wallet_accounts_chain_network_keyset
  ON app.wallet_accounts (chain, network, keyset_id);

ALTER TABLE app.wallet_accounts
  DROP CONSTRAINT IF EXISTS wallet_accounts_key_material_hash_requires_algo;
ALTER TABLE app.wallet_accounts
  ADD CONSTRAINT wallet_accounts_key_material_hash_requires_algo
  CHECK (
    (key_material_hash IS NULL AND key_material_hash_algo IS NULL)
    OR (key_material_hash IS NOT NULL AND key_material_hash_algo = 'hmac-sha256')
  );

ALTER TABLE app.wallet_accounts
  DROP CONSTRAINT IF EXISTS wallet_accounts_key_material_hash_hex;
ALTER TABLE app.wallet_accounts
  ADD CONSTRAINT wallet_accounts_key_material_hash_hex
  CHECK (
    key_material_hash IS NULL
    OR key_material_hash ~ '^[0-9a-f]{64}$'
  );
