#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd docker
require_cmd jq
require_cmd openssl

ETH_ARTIFACT_FILE="${ETH_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/eth.json}"
SERVICE_DB_USER="${SERVICE_DB_USER:-chaintx}"
SERVICE_DB_NAME="${SERVICE_DB_NAME:-chaintx}"
SERVICE_SYNC_TIMEOUT_SECONDS="${SERVICE_SYNC_TIMEOUT_SECONDS:-120}"

LOCAL_ETH_KEYSET_ID="${LOCAL_ETH_KEYSET_ID:-ks_eth_local}"
LOCAL_ETH_NETWORK="${LOCAL_ETH_NETWORK:-local}"
LOCAL_SYNC_SOURCE_REF="${LOCAL_SYNC_SOURCE_REF:-local-artifact:service_sync_catalog.sh}"

BTC_REGTEST_KEYSET_ID="${BTC_REGTEST_KEYSET_ID:-ks_btc_regtest}"
BTC_REGTEST_NETWORK="${BTC_REGTEST_NETWORK:-regtest}"
BTC_TESTNET_KEYSET_ID="${BTC_TESTNET_KEYSET_ID:-ks_btc_testnet}"
BTC_TESTNET_NETWORK="${BTC_TESTNET_NETWORK:-testnet}"

SEPOLIA_KEYSET_ID="${SEPOLIA_KEYSET_ID:-ks_eth_sepolia}"
SEPOLIA_NETWORK="${SEPOLIA_NETWORK:-sepolia}"
SEPOLIA_CHAIN_ID="${SEPOLIA_CHAIN_ID:-11155111}"
SEPOLIA_USDT_CONTRACT="${SEPOLIA_USDT_CONTRACT:-0x1c7d4b196cb0c7b01d743fbc6116a902379c7238}"
SEPOLIA_SYNC_SOURCE_REF="${SEPOLIA_SYNC_SOURCE_REF:-seed-default:service_sync_catalog.sh}"

KEYSET_HASH_ALGO="hmac-sha256"
DEVTEST_KEYSETS_JSON="${PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON:-}"
KEYSET_HASH_HMAC_SECRET="${PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET:-}"
KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON="${PAYMENT_REQUEST_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON:-}"

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
    local has_catalog has_wallet has_wallet_hash_columns
    has_catalog="$(pg_scalar "SELECT to_regclass('app.asset_catalog') IS NOT NULL;")"
    has_wallet="$(pg_scalar "SELECT to_regclass('app.wallet_accounts') IS NOT NULL;")"
    has_wallet_hash_columns="$(pg_scalar "
SELECT count(*)
FROM information_schema.columns
WHERE table_schema = 'app'
  AND table_name = 'wallet_accounts'
  AND column_name IN ('key_material_hash', 'key_material_hash_algo');
")"

    if [ "$has_catalog" = "t" ] && [ "$has_wallet" = "t" ] && [ "$has_wallet_hash_columns" -ge 2 ]; then
      return 0
    fi
    sleep 2
  done

  echo "timed out waiting for app catalog tables and wallet hash columns to become available" >&2
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

sql_escape() {
  printf '%s' "$1" | sed "s/'/''/g"
}

bind_asset_wallet_account() {
  local chain="$1"
  local network="$2"
  local asset="$3"
  local wallet_account_id="$4"
  local address_scheme="$5"

  local chain_sql network_sql asset_sql wallet_account_id_sql address_scheme_sql
  chain_sql="$(sql_escape "$chain")"
  network_sql="$(sql_escape "$network")"
  asset_sql="$(sql_escape "$asset")"
  wallet_account_id_sql="$(sql_escape "$wallet_account_id")"
  address_scheme_sql="$(sql_escape "$address_scheme")"

  local updated_count
  updated_count="$(pg_scalar "
WITH updated AS (
  UPDATE app.asset_catalog
  SET
    wallet_account_id = '$wallet_account_id_sql',
    address_scheme = '$address_scheme_sql',
    enabled = TRUE,
    updated_at = now()
  WHERE chain = '$chain_sql'
    AND network = '$network_sql'
    AND asset = '$asset_sql'
  RETURNING 1
)
SELECT count(*) FROM updated;
")"

  if [ "$updated_count" -eq 0 ]; then
    echo "missing asset_catalog row for chain=$chain network=$network asset=$asset while binding wallet_account_id=$wallet_account_id" >&2
    exit 1
  fi
}

require_keyset_hash_config() {
  if [ -z "$DEVTEST_KEYSETS_JSON" ]; then
    echo "missing PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON for wallet-account hash sync" >&2
    exit 1
  fi
  if ! printf '%s' "$DEVTEST_KEYSETS_JSON" | jq -e 'type == "object"' >/dev/null 2>&1; then
    echo "invalid PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON: must be a JSON object" >&2
    exit 1
  fi
  if [ -z "$KEYSET_HASH_HMAC_SECRET" ]; then
    echo "missing PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET for wallet-account hash sync" >&2
    exit 1
  fi
  if [ -n "$KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON" ] && ! printf '%s' "$KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON" | jq -e 'type == "array"' >/dev/null 2>&1; then
    echo "invalid PAYMENT_REQUEST_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON: must be a JSON array" >&2
    exit 1
  fi
}

resolve_keyset_material() {
  local chain="$1"
  local network="$2"
  local keyset_id="$3"
  local key_material
  key_material="$(
    printf '%s' "$DEVTEST_KEYSETS_JSON" | jq -r --arg keyset_id "$keyset_id" '
      .[$keyset_id] as $legacy |
      if $legacy == null then ""
      elif ($legacy | type) == "string" then $legacy
      elif ($legacy | type) == "object" then ($legacy.extended_public_key // $legacy.key_material // $legacy.xpub // "")
      else ""
      end
    '
  )"

  if [ -z "$key_material" ] || [ "$key_material" = "null" ]; then
    key_material="$(
      printf '%s' "$DEVTEST_KEYSETS_JSON" | jq -r --arg chain "$chain" --arg network "$network" --arg keyset_id "$keyset_id" '
        .[$chain][$network] as $entry |
        if $entry == null then ""
        elif ($entry | type) != "object" then ""
        elif (($entry.keyset_id // "") != $keyset_id) then ""
        else ($entry.extended_public_key // $entry.key_material // $entry.xpub // "")
        end
      '
    )"
  fi

  if [ -z "$key_material" ] || [ "$key_material" = "null" ]; then
    echo "missing key material for chain=$chain network=$network keyset_id=$keyset_id in PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON" >&2
    exit 1
  fi

  printf '%s' "$key_material"
}

compute_key_material_hash_for_secret() {
  local key_material="$1"
  local secret="$2"
  local digest
  digest="$(printf '%s' "$key_material" | openssl dgst -sha256 -hmac "$secret" -r | awk '{print $1}')"
  if ! printf '%s' "$digest" | grep -Eq '^[0-9a-f]{64}$'; then
    echo "failed to compute valid hmac-sha256 digest" >&2
    exit 1
  fi
  printf '%s' "$digest"
}

build_key_hash_candidates() {
  local key_material="$1"
  local active_hash
  active_hash="$(compute_key_material_hash_for_secret "$key_material" "$KEYSET_HASH_HMAC_SECRET")"
  printf 'active|%s\n' "$active_hash"

  if [ -z "$KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON" ]; then
    return 0
  fi

  while IFS= read -r legacy_secret; do
    [ -n "$legacy_secret" ] || continue
    local legacy_hash
    legacy_hash="$(compute_key_material_hash_for_secret "$key_material" "$legacy_secret")"
    if [ "$legacy_hash" = "$active_hash" ]; then
      continue
    fi
    printf 'legacy|%s\n' "$legacy_hash"
  done < <(printf '%s' "$KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON" | jq -r '.[] | strings | select(length > 0)')
}

generate_wallet_account_id() {
  local chain="$1"
  local network="$2"
  local key_hash="$3"
  local timestamp
  timestamp="$(date -u +%Y%m%d%H%M%S)"
  printf 'wa_%s_%s_%s_%04d_%s' "$chain" "$network" "${key_hash:0:12}" "$RANDOM" "$timestamp"
}

ensure_wallet_account_for_keyset() {
  local chain="$1"
  local network="$2"
  local keyset_id="$3"

  local key_material key_hash candidate_hashes
  key_material="$(resolve_keyset_material "$chain" "$network" "$keyset_id")"
  candidate_hashes="$(build_key_hash_candidates "$key_material")"
  key_hash="$(printf '%s\n' "$candidate_hashes" | head -n1 | awk -F'|' '{print $2}')"
  if [ -z "$key_hash" ]; then
    echo "failed to build key hash candidates for chain=$chain network=$network keyset_id=$keyset_id" >&2
    exit 1
  fi
  local key_hash_sql
  key_hash_sql="$(sql_escape "$key_hash")"

  local chain_sql network_sql keyset_sql
  chain_sql="$(sql_escape "$chain")"
  network_sql="$(sql_escape "$network")"
  keyset_sql="$(sql_escape "$keyset_id")"

  local active_row active_id active_hash
  active_row="$(pg_scalar "
SELECT id || '|' || COALESCE(key_material_hash, '')
FROM app.wallet_accounts
WHERE chain = '$chain_sql'
  AND network = '$network_sql'
  AND keyset_id = '$keyset_sql'
  AND is_active = TRUE
ORDER BY updated_at DESC, created_at DESC
LIMIT 1;
")"

  active_id=""
  active_hash=""
  if [ -n "$active_row" ]; then
    active_id="${active_row%%|*}"
    active_hash="${active_row#*|}"
  fi

  local active_match_source
  active_match_source=""
  if [ -n "$active_id" ]; then
    if [ -z "$active_hash" ]; then
      active_match_source="unhashed"
    elif [ "$active_hash" = "$key_hash" ]; then
      active_match_source="active"
    else
      while IFS= read -r candidate; do
        [ -n "$candidate" ] || continue
        local source hash
        source="${candidate%%|*}"
        hash="${candidate#*|}"
        if [ "$source" = "legacy" ] && [ "$active_hash" = "$hash" ]; then
          active_match_source="legacy"
          break
        fi
      done <<< "$candidate_hashes"
    fi
  fi

  if [ -n "$active_id" ] && [ -n "$active_match_source" ]; then
    local active_id_sql
    active_id_sql="$(sql_escape "$active_id")"

    pg_exec "
UPDATE app.wallet_accounts
SET
  derivation_path_template = '0/{index}',
  key_material_hash = '$key_hash_sql',
  key_material_hash_algo = '$KEYSET_HASH_ALGO',
  is_active = TRUE,
  updated_at = now()
WHERE id = '$active_id_sql';
"

    echo "wallet-account sync action=reused chain=$chain network=$network keyset_id=$keyset_id wallet_account_id=$active_id hash_prefix=${key_hash:0:12} match_source=$active_match_source" >&2
    printf '%s' "$active_id"
    return 0
  fi

  local historical_id historical_match_source
  historical_id=""
  historical_match_source=""
  while IFS= read -r candidate; do
    [ -n "$candidate" ] || continue
    local source hash
    source="${candidate%%|*}"
    hash="${candidate#*|}"
    [ -n "$hash" ] || continue

    local hash_sql
    hash_sql="$(sql_escape "$hash")"
    historical_id="$(pg_scalar "
SELECT id
FROM app.wallet_accounts
WHERE chain = '$chain_sql'
  AND network = '$network_sql'
  AND keyset_id = '$keyset_sql'
  AND is_active = FALSE
  AND key_material_hash = '$hash_sql'
ORDER BY updated_at DESC, created_at DESC
LIMIT 1;
")"
    if [ -n "$historical_id" ]; then
      historical_match_source="$source"
      break
    fi
  done <<< "$candidate_hashes"

  if [ -n "$historical_id" ]; then
    local historical_id_sql
    historical_id_sql="$(sql_escape "$historical_id")"

    pg_exec "
BEGIN;
UPDATE app.wallet_accounts
SET
  is_active = FALSE,
  updated_at = now()
WHERE chain = '$chain_sql'
  AND network = '$network_sql'
  AND keyset_id = '$keyset_sql'
  AND is_active = TRUE;

UPDATE app.wallet_accounts
SET
  derivation_path_template = '0/{index}',
  key_material_hash = '$key_hash_sql',
  key_material_hash_algo = '$KEYSET_HASH_ALGO',
  is_active = TRUE,
  updated_at = now()
WHERE id = '$historical_id_sql';
COMMIT;
"

    echo "wallet-account sync action=reactivated chain=$chain network=$network keyset_id=$keyset_id wallet_account_id=$historical_id hash_prefix=${key_hash:0:12} match_source=${historical_match_source:-active}" >&2
    printf '%s' "$historical_id"
    return 0
  fi

  local new_wallet_account_id
  new_wallet_account_id="$(generate_wallet_account_id "$chain" "$network" "$key_hash")"

  local new_id_sql
  new_id_sql="$(sql_escape "$new_wallet_account_id")"

  pg_exec "
BEGIN;
UPDATE app.wallet_accounts
SET
  is_active = FALSE,
  updated_at = now()
WHERE chain = '$chain_sql'
  AND network = '$network_sql'
  AND keyset_id = '$keyset_sql'
  AND is_active = TRUE;

INSERT INTO app.wallet_accounts (
  id,
  chain,
  network,
  keyset_id,
  derivation_path_template,
  next_index,
  is_active,
  key_material_hash,
  key_material_hash_algo,
  created_at,
  updated_at
)
VALUES (
  '$new_id_sql',
  '$chain_sql',
  '$network_sql',
  '$keyset_sql',
  '0/{index}',
  0,
  TRUE,
  '$key_hash_sql',
  '$KEYSET_HASH_ALGO',
  now(),
  now()
);
COMMIT;
"

  echo "wallet-account sync action=rotated chain=$chain network=$network keyset_id=$keyset_id wallet_account_id=$new_wallet_account_id hash_prefix=${key_hash:0:12} match_source=active" >&2
  printf '%s' "$new_wallet_account_id"
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
require_keyset_hash_config

btc_regtest_wallet_account_id="$(ensure_wallet_account_for_keyset "bitcoin" "$BTC_REGTEST_NETWORK" "$BTC_REGTEST_KEYSET_ID")"
bind_asset_wallet_account "bitcoin" "$BTC_REGTEST_NETWORK" "BTC" "$btc_regtest_wallet_account_id" "bip84_p2wpkh"

btc_testnet_wallet_account_id="$(ensure_wallet_account_for_keyset "bitcoin" "$BTC_TESTNET_NETWORK" "$BTC_TESTNET_KEYSET_ID")"
bind_asset_wallet_account "bitcoin" "$BTC_TESTNET_NETWORK" "BTC" "$btc_testnet_wallet_account_id" "bip84_p2wpkh"

sepolia_wallet_account_id="$(ensure_wallet_account_for_keyset "ethereum" "$SEPOLIA_NETWORK" "$SEPOLIA_KEYSET_ID")"
local_wallet_account_id="$(ensure_wallet_account_for_keyset "ethereum" "$LOCAL_ETH_NETWORK" "$LOCAL_ETH_KEYSET_ID")"

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
  '$sepolia_wallet_account_id',
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
  '$sepolia_wallet_account_id',
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
  echo "service catalog synced: bitcoin/$BTC_REGTEST_NETWORK wallet_account_id=$btc_regtest_wallet_account_id + bitcoin/$BTC_TESTNET_NETWORK wallet_account_id=$btc_testnet_wallet_account_id + ethereum/$SEPOLIA_NETWORK wallet_account_id=$sepolia_wallet_account_id (ethereum/$LOCAL_ETH_NETWORK artifacts missing; wallet-account hash ensured with id=$local_wallet_account_id)"
  exit 0
fi

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
  '$local_wallet_account_id',
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
  '$local_wallet_account_id',
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

echo "service catalog synced: bitcoin/$BTC_REGTEST_NETWORK wallet_account_id=$btc_regtest_wallet_account_id + bitcoin/$BTC_TESTNET_NETWORK wallet_account_id=$btc_testnet_wallet_account_id + ethereum/$SEPOLIA_NETWORK wallet_account_id=$sepolia_wallet_account_id + ethereum/$LOCAL_ETH_NETWORK wallet_account_id=$local_wallet_account_id"
