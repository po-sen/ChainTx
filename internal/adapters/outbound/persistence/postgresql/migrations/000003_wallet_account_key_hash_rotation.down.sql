DROP INDEX IF EXISTS idx_wallet_accounts_chain_network_keyset_active;
DROP INDEX IF EXISTS idx_wallet_accounts_chain_network_keyset;

ALTER TABLE app.wallet_accounts
  DROP CONSTRAINT IF EXISTS wallet_accounts_key_material_hash_requires_algo;
ALTER TABLE app.wallet_accounts
  DROP CONSTRAINT IF EXISTS wallet_accounts_key_material_hash_hex;

ALTER TABLE app.wallet_accounts
  DROP COLUMN IF EXISTS key_material_hash_algo,
  DROP COLUMN IF EXISTS key_material_hash;

ALTER TABLE app.wallet_accounts
  ADD CONSTRAINT wallet_accounts_chain_network_keyset_unique UNIQUE (chain, network, keyset_id);
