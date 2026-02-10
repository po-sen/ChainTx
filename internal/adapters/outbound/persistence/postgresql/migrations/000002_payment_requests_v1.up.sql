CREATE TABLE IF NOT EXISTS app.wallet_accounts (
  id text PRIMARY KEY,
  chain text NOT NULL,
  network text NOT NULL,
  keyset_id text NOT NULL,
  derivation_path_template text NOT NULL,
  address_scheme text NOT NULL,
  chain_id bigint,
  next_index bigint NOT NULL DEFAULT 0,
  is_active boolean NOT NULL DEFAULT TRUE,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT wallet_accounts_next_index_non_negative CHECK (next_index >= 0),
  CONSTRAINT wallet_accounts_chain_network_keyset_unique UNIQUE (chain, network, keyset_id)
);

CREATE TABLE IF NOT EXISTS app.asset_catalog (
  chain text NOT NULL,
  network text NOT NULL,
  asset text NOT NULL,
  wallet_account_id text NOT NULL REFERENCES app.wallet_accounts (id),
  minor_unit text NOT NULL,
  decimals integer NOT NULL,
  address_scheme text NOT NULL,
  chain_id bigint,
  token_standard text,
  token_contract text,
  token_decimals integer,
  default_expires_in_seconds bigint NOT NULL,
  enabled boolean NOT NULL DEFAULT TRUE,
  source_ref text,
  approved_by text,
  approved_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (chain, network, asset),
  CONSTRAINT asset_catalog_decimals_non_negative CHECK (decimals >= 0),
  CONSTRAINT asset_catalog_default_expires_range CHECK (
    default_expires_in_seconds >= 60 AND default_expires_in_seconds <= 2592000
  )
);

CREATE TABLE IF NOT EXISTS app.payment_requests (
  id text PRIMARY KEY,
  wallet_account_id text NOT NULL REFERENCES app.wallet_accounts (id),
  chain text NOT NULL,
  network text NOT NULL,
  asset text NOT NULL,
  status text NOT NULL,
  expected_amount_minor numeric(78,0),
  address_canonical text NOT NULL,
  address_scheme text NOT NULL,
  derivation_index bigint NOT NULL,
  chain_id bigint,
  token_standard text,
  token_contract text,
  token_decimals integer,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT payment_requests_status_non_empty CHECK (status <> ''),
  CONSTRAINT payment_requests_derivation_index_non_negative CHECK (derivation_index >= 0),
  CONSTRAINT payment_requests_expected_amount_non_negative CHECK (
    expected_amount_minor IS NULL OR expected_amount_minor >= 0
  ),
  CONSTRAINT payment_requests_metadata_size CHECK (octet_length(metadata::text) <= 4096),
  CONSTRAINT payment_requests_expiry_after_create CHECK (expires_at > created_at),
  CONSTRAINT payment_requests_expiry_minimum CHECK (expires_at >= created_at + interval '60 seconds'),
  CONSTRAINT payment_requests_expiry_maximum CHECK (expires_at <= created_at + interval '30 days'),
  CONSTRAINT payment_requests_wallet_index_unique UNIQUE (wallet_account_id, derivation_index),
  CONSTRAINT payment_requests_chain_network_address_unique UNIQUE (chain, network, address_canonical)
);

CREATE TABLE IF NOT EXISTS app.idempotency_records (
  scope_principal text NOT NULL,
  scope_method text NOT NULL,
  scope_path text NOT NULL,
  idempotency_key text NOT NULL,
  request_hash text NOT NULL,
  hash_algorithm text NOT NULL,
  resource_id text NOT NULL,
  response_payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  PRIMARY KEY (scope_principal, scope_method, scope_path, idempotency_key),
  CONSTRAINT idempotency_records_expires_minimum CHECK (
    expires_at >= created_at + interval '24 hours'
  )
);

CREATE INDEX IF NOT EXISTS idx_payment_requests_created_at ON app.payment_requests (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_requests_wallet ON app.payment_requests (wallet_account_id);
CREATE INDEX IF NOT EXISTS idx_asset_catalog_enabled ON app.asset_catalog (enabled, chain, network, asset);

INSERT INTO app.wallet_accounts (
  id,
  chain,
  network,
  keyset_id,
  derivation_path_template,
  address_scheme,
  chain_id,
  next_index,
  is_active
)
VALUES
  (
    'wa_btc_mainnet_001',
    'bitcoin',
    'mainnet',
    'btc-mainnet-hot',
    'm/84''/0''/0''/0/{index}',
    'bip84_p2wpkh',
    NULL,
    0,
    TRUE
  ),
  (
    'wa_eth_mainnet_001',
    'ethereum',
    'mainnet',
    'eth-mainnet-hot',
    'm/44''/60''/0''/0/{index}',
    'evm_bip44',
    1,
    0,
    TRUE
  )
ON CONFLICT (id) DO UPDATE
SET
  chain = EXCLUDED.chain,
  network = EXCLUDED.network,
  keyset_id = EXCLUDED.keyset_id,
  derivation_path_template = EXCLUDED.derivation_path_template,
  address_scheme = EXCLUDED.address_scheme,
  chain_id = EXCLUDED.chain_id,
  is_active = EXCLUDED.is_active,
  updated_at = now();

INSERT INTO app.asset_catalog (
  chain,
  network,
  asset,
  wallet_account_id,
  minor_unit,
  decimals,
  address_scheme,
  chain_id,
  token_standard,
  token_contract,
  token_decimals,
  default_expires_in_seconds,
  enabled,
  source_ref,
  approved_by,
  approved_at
)
VALUES
  (
    'bitcoin',
    'mainnet',
    'BTC',
    'wa_btc_mainnet_001',
    'sats',
    8,
    'bip84_p2wpkh',
    NULL,
    NULL,
    NULL,
    NULL,
    3600,
    TRUE,
    'spec:2026-02-09-multi-asset-payment-request-api',
    'system',
    now()
  ),
  (
    'ethereum',
    'mainnet',
    'ETH',
    'wa_eth_mainnet_001',
    'wei',
    18,
    'evm_bip44',
    1,
    NULL,
    NULL,
    NULL,
    3600,
    TRUE,
    'spec:2026-02-09-multi-asset-payment-request-api',
    'system',
    now()
  ),
  (
    'ethereum',
    'mainnet',
    'USDT',
    'wa_eth_mainnet_001',
    'token_minor',
    6,
    'evm_bip44',
    1,
    'ERC20',
    '0xdac17f958d2ee523a2206206994597c13d831ec7',
    6,
    3600,
    TRUE,
    'spec:2026-02-09-multi-asset-payment-request-api',
    'system',
    now()
  )
ON CONFLICT (chain, network, asset) DO UPDATE
SET
  wallet_account_id = EXCLUDED.wallet_account_id,
  minor_unit = EXCLUDED.minor_unit,
  decimals = EXCLUDED.decimals,
  address_scheme = EXCLUDED.address_scheme,
  chain_id = EXCLUDED.chain_id,
  token_standard = EXCLUDED.token_standard,
  token_contract = EXCLUDED.token_contract,
  token_decimals = EXCLUDED.token_decimals,
  default_expires_in_seconds = EXCLUDED.default_expires_in_seconds,
  enabled = EXCLUDED.enabled,
  source_ref = EXCLUDED.source_ref,
  approved_by = EXCLUDED.approved_by,
  approved_at = EXCLUDED.approved_at,
  updated_at = now();
