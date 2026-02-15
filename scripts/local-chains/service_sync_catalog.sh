#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd docker
require_cmd jq

ETH_ARTIFACT_FILE="${ETH_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/eth.json}"
SERVICE_DB_USER="${SERVICE_DB_USER:-chaintx}"
SERVICE_DB_NAME="${SERVICE_DB_NAME:-chaintx}"
SERVICE_SYNC_TIMEOUT_SECONDS="${SERVICE_SYNC_TIMEOUT_SECONDS:-120}"

LOCAL_ETH_WALLET_ACCOUNT_ID="${LOCAL_ETH_WALLET_ACCOUNT_ID:-wa_eth_local_001}"
LOCAL_ETH_KEYSET_ID="${LOCAL_ETH_KEYSET_ID:-ks_eth_local}"
LOCAL_ETH_NETWORK="${LOCAL_ETH_NETWORK:-local}"
LOCAL_SYNC_SOURCE_REF="${LOCAL_SYNC_SOURCE_REF:-local-artifact:service_sync_catalog.sh}"

SEPOLIA_WALLET_ACCOUNT_ID="${SEPOLIA_WALLET_ACCOUNT_ID:-wa_eth_sepolia_001}"
SEPOLIA_KEYSET_ID="${SEPOLIA_KEYSET_ID:-ks_eth_sepolia}"
SEPOLIA_NETWORK="${SEPOLIA_NETWORK:-sepolia}"
SEPOLIA_CHAIN_ID="${SEPOLIA_CHAIN_ID:-11155111}"
SEPOLIA_USDT_CONTRACT="${SEPOLIA_USDT_CONTRACT:-0x1c7d4b196cb0c7b01d743fbc6116a902379c7238}"
SEPOLIA_SYNC_SOURCE_REF="${SEPOLIA_SYNC_SOURCE_REF:-seed-default:service_sync_catalog.sh}"

svc_dc() {
  dc "$SERVICE_COMPOSE_FILE" "$LOCAL_SERVICE_PROJECT" "$@"
}

pg_exec() {
  local sql="$1"
  svc_dc exec -T postgres psql -v ON_ERROR_STOP=1 -U "$SERVICE_DB_USER" -d "$SERVICE_DB_NAME" -q -c "$sql" >/dev/null
}

pg_scalar() {
  local sql="$1"
  svc_dc exec -T postgres psql -v ON_ERROR_STOP=1 -U "$SERVICE_DB_USER" -d "$SERVICE_DB_NAME" -t -A -c "$sql" | tr -d '\r'
}

wait_for_postgres() {
  local deadline=$((SECONDS + SERVICE_SYNC_TIMEOUT_SECONDS))
  while (( SECONDS < deadline )); do
    if svc_dc exec -T postgres pg_isready -U "$SERVICE_DB_USER" -d "$SERVICE_DB_NAME" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done

  echo "timed out waiting for service postgres readiness" >&2
  return 1
}

wait_for_app_tables() {
  local deadline=$((SECONDS + SERVICE_SYNC_TIMEOUT_SECONDS))
  while (( SECONDS < deadline )); do
    local has_catalog has_wallet
    has_catalog="$(pg_scalar "SELECT to_regclass('app.asset_catalog') IS NOT NULL;")"
    has_wallet="$(pg_scalar "SELECT to_regclass('app.wallet_accounts') IS NOT NULL;")"

    if [ "$has_catalog" = "t" ] && [ "$has_wallet" = "t" ]; then
      return 0
    fi
    sleep 2
  done

  echo "timed out waiting for app catalog tables to become available" >&2
  return 1
}

validate_chain_id() {
  local raw="$1"
  if [ -z "$raw" ] || ! printf '%s' "$raw" | grep -Eq '^[0-9]+$'; then
    return 1
  fi
}

validate_evm_address() {
  local raw="$1"
  if [ -z "$raw" ] || ! printf '%s' "$raw" | grep -Eq '^0x[0-9a-fA-F]{40}$'; then
    return 1
  fi
}

eth_chain_id=""
usdt_contract=""
usdt_decimals=""
if [ -f "$ETH_ARTIFACT_FILE" ]; then
  eth_chain_id="$(jq -r '.chain_id // empty' "$ETH_ARTIFACT_FILE")"
  usdt_contract="$(jq -r '.usdt_contract_address // empty' "$ETH_ARTIFACT_FILE")"
  usdt_decimals="$(jq -r '.usdt_token_decimals // empty' "$ETH_ARTIFACT_FILE")"

  if ! validate_chain_id "$eth_chain_id"; then
    echo "invalid eth artifact chain_id in $ETH_ARTIFACT_FILE" >&2
    exit 1
  fi

  if [ -n "$usdt_contract" ] || [ -n "$usdt_decimals" ]; then
    if ! validate_evm_address "$usdt_contract"; then
      echo "invalid eth artifact usdt_contract_address in $ETH_ARTIFACT_FILE" >&2
      exit 1
    fi
    if ! validate_chain_id "$usdt_decimals"; then
      echo "invalid eth artifact usdt_token_decimals in $ETH_ARTIFACT_FILE" >&2
      exit 1
    fi
  fi
fi

has_local_artifacts=1
if [ -z "$eth_chain_id" ]; then
  has_local_artifacts=0
fi

local_chain_id="$eth_chain_id"

wait_for_postgres
wait_for_app_tables

pg_exec "
INSERT INTO app.wallet_accounts (
  id,
  chain,
  network,
  keyset_id,
  derivation_path_template,
  next_index,
  is_active,
  created_at,
  updated_at
)
VALUES (
  '$SEPOLIA_WALLET_ACCOUNT_ID',
  'ethereum',
  '$SEPOLIA_NETWORK',
  '$SEPOLIA_KEYSET_ID',
  '0/{index}',
  0,
  TRUE,
  now(),
  now()
)
ON CONFLICT (id) DO UPDATE
SET
  chain = EXCLUDED.chain,
  network = EXCLUDED.network,
  keyset_id = EXCLUDED.keyset_id,
  derivation_path_template = EXCLUDED.derivation_path_template,
  is_active = EXCLUDED.is_active,
  updated_at = now();
"

pg_exec "
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
  approved_at,
  created_at,
  updated_at
)
VALUES (
  'ethereum',
  '$SEPOLIA_NETWORK',
  'ETH',
  '$SEPOLIA_WALLET_ACCOUNT_ID',
  'wei',
  18,
  'evm_bip44',
  $SEPOLIA_CHAIN_ID,
  NULL,
  NULL,
  NULL,
  3600,
  TRUE,
  '$SEPOLIA_SYNC_SOURCE_REF',
  'system',
  now(),
  now(),
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
"

pg_exec "
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
  approved_at,
  created_at,
  updated_at
)
VALUES (
  'ethereum',
  '$SEPOLIA_NETWORK',
  'USDT',
  '$SEPOLIA_WALLET_ACCOUNT_ID',
  'token_minor',
  6,
  'evm_bip44',
  $SEPOLIA_CHAIN_ID,
  'ERC20',
  '$SEPOLIA_USDT_CONTRACT',
  6,
  3600,
  TRUE,
  '$SEPOLIA_SYNC_SOURCE_REF',
  'system',
  now(),
  now(),
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
"

if [ "$has_local_artifacts" -eq 0 ]; then
  echo "service catalog synced: ethereum/$SEPOLIA_NETWORK baseline only (no local artifacts)"
  exit 0
fi

pg_exec "
INSERT INTO app.wallet_accounts (
  id,
  chain,
  network,
  keyset_id,
  derivation_path_template,
  next_index,
  is_active,
  created_at,
  updated_at
)
VALUES (
  '$LOCAL_ETH_WALLET_ACCOUNT_ID',
  'ethereum',
  '$LOCAL_ETH_NETWORK',
  '$LOCAL_ETH_KEYSET_ID',
  '0/{index}',
  0,
  TRUE,
  now(),
  now()
)
ON CONFLICT (id) DO UPDATE
SET
  chain = EXCLUDED.chain,
  network = EXCLUDED.network,
  keyset_id = EXCLUDED.keyset_id,
  derivation_path_template = EXCLUDED.derivation_path_template,
  is_active = EXCLUDED.is_active,
  updated_at = now();
"

pg_exec "
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
  approved_at,
  created_at,
  updated_at
)
VALUES (
  'ethereum',
  '$LOCAL_ETH_NETWORK',
  'ETH',
  '$LOCAL_ETH_WALLET_ACCOUNT_ID',
  'wei',
  18,
  'evm_bip44',
  $local_chain_id,
  NULL,
  NULL,
  NULL,
  3600,
  TRUE,
  '$LOCAL_SYNC_SOURCE_REF',
  'system',
  now(),
  now(),
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
"

if [ -n "$usdt_contract" ]; then
  pg_exec "
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
  approved_at,
  created_at,
  updated_at
)
VALUES (
  'ethereum',
  '$LOCAL_ETH_NETWORK',
  'USDT',
  '$LOCAL_ETH_WALLET_ACCOUNT_ID',
  'token_minor',
  6,
  'evm_bip44',
  $local_chain_id,
  'ERC20',
  '$usdt_contract',
  $usdt_decimals,
  3600,
  TRUE,
  '$LOCAL_SYNC_SOURCE_REF',
  'system',
  now(),
  now(),
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
"
fi

echo "service catalog synced: ethereum/$SEPOLIA_NETWORK baseline + ethereum/$LOCAL_ETH_NETWORK local profile"
