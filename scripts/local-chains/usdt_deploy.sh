#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd cast
require_cmd forge

ensure_artifact_dir

ETH_ARTIFACT_FILE="${ETH_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/eth.json}"
CONTRACT_ROOT="${CONTRACT_ROOT:-$REPO_ROOT/scripts/local-chains/contracts}"
USDT_MINT_AMOUNT="${USDT_MINT_AMOUNT:-1000000000000}"
USDT_MINT_TO="${USDT_MINT_TO:-$ANVIL_DEFAULT_ADDRESS}"
PAYER_PRIVATE_KEY="${PAYER_PRIVATE_KEY:-$ANVIL_DEFAULT_PRIVATE_KEY}"
PAYER_ADDRESS="${PAYER_ADDRESS:-$ANVIL_DEFAULT_ADDRESS}"
USDT_TOKEN_DECIMALS="${USDT_TOKEN_DECIMALS:-6}"

json_get_raw() {
  local key="$1"
  local file="$2"

  awk -v target="\"${key}\"" '
    $0 ~ "^[[:space:]]*" target "[[:space:]]*:" {
      raw = $0
      sub(/^[^:]*:[[:space:]]*/, "", raw)
      sub(/[[:space:]]*,[[:space:]]*$/, "", raw)
      print raw
      exit
    }
  ' "$file"
}

json_get_string() {
  local key="$1"
  local file="$2"
  local raw

  raw="$(json_get_raw "$key" "$file")"
  raw="${raw#\"}"
  raw="${raw%\"}"
  printf '%s' "$raw"
}

json_get_number() {
  local key="$1"
  local file="$2"
  local raw

  raw="$(json_get_raw "$key" "$file")"
  raw="${raw#\"}"
  raw="${raw%\"}"
  printf '%s' "$raw"
}

normalize_nullable_string() {
  local raw="$1"
  if [ "$raw" = "null" ]; then
    printf ''
    return 0
  fi
  printf '%s' "$raw"
}

is_uint() {
  printf '%s' "$1" | grep -Eq '^[0-9]+$'
}

is_hex_hash() {
  printf '%s' "$1" | grep -Eq '^0x[0-9a-fA-F]{64}$'
}

is_evm_address() {
  printf '%s' "$1" | grep -Eq '^0x[0-9a-fA-F]{40}$'
}

actual_chain_id="$(cast chain-id --rpc-url "$ETH_RPC_URL" 2>/dev/null | tr -d '\r')"
if [ -z "$actual_chain_id" ] || ! is_uint "$actual_chain_id"; then
  echo "cannot reach ETH RPC at $ETH_RPC_URL (run make chain-up-eth first)" >&2
  exit 1
fi

actual_genesis="$(cast block 0 --rpc-url "$ETH_RPC_URL" --field hash 2>/dev/null | tr -d '\r')"
if [ -z "$actual_genesis" ] || ! is_hex_hash "$actual_genesis"; then
  echo "failed to query ETH genesis block from $ETH_RPC_URL" >&2
  exit 1
fi

if [ "$actual_chain_id" != "$ETH_EXPECTED_CHAIN_ID" ]; then
  echo "unexpected chain id from ETH RPC: got $actual_chain_id expected $ETH_EXPECTED_CHAIN_ID" >&2
  exit 1
fi

if [ ! -f "$ETH_ARTIFACT_FILE" ]; then
  echo "missing ETH artifact: $ETH_ARTIFACT_FILE (run make chain-up-eth first)" >&2
  exit 1
fi

artifact_chain_id="$(json_get_number chain_id "$ETH_ARTIFACT_FILE")"
artifact_genesis="$(json_get_string genesis_block_hash "$ETH_ARTIFACT_FILE")"
artifact_rpc_url="$(normalize_nullable_string "$(json_get_string rpc_url "$ETH_ARTIFACT_FILE")")"
artifact_compose_project="$(normalize_nullable_string "$(json_get_string compose_project "$ETH_ARTIFACT_FILE")")"
artifact_receiver_xpub="$(normalize_nullable_string "$(json_get_string receiver_xpub "$ETH_ARTIFACT_FILE")")"
artifact_payer_private_key="$(normalize_nullable_string "$(json_get_string payer_private_key "$ETH_ARTIFACT_FILE")")"
artifact_payer_address="$(normalize_nullable_string "$(json_get_string payer_address "$ETH_ARTIFACT_FILE")")"
existing_contract="$(normalize_nullable_string "$(json_get_string usdt_contract_address "$ETH_ARTIFACT_FILE")")"

if ! is_uint "$artifact_chain_id" || [ "$artifact_chain_id" != "$actual_chain_id" ] || [ "$artifact_genesis" != "$actual_genesis" ]; then
  echo "stale eth artifact fingerprint detected. run: make chain-down-eth && make chain-up-eth" >&2
  exit 1
fi

if [ -z "$artifact_rpc_url" ]; then
  artifact_rpc_url="$ETH_RPC_URL"
fi

if [ -z "$artifact_compose_project" ]; then
  artifact_compose_project="$LOCAL_ETH_PROJECT"
fi

if [ -z "$artifact_receiver_xpub" ]; then
  echo "invalid eth artifact: missing receiver_xpub in $ETH_ARTIFACT_FILE" >&2
  exit 1
fi

if [ -z "$artifact_payer_private_key" ]; then
  artifact_payer_private_key="$PAYER_PRIVATE_KEY"
fi

if [ -z "$artifact_payer_address" ]; then
  artifact_payer_address="$PAYER_ADDRESS"
fi

if is_evm_address "$existing_contract" && [ "$existing_contract" != "0x0000000000000000000000000000000000000000" ]; then
  existing_code="$(cast code --rpc-url "$ETH_RPC_URL" "$existing_contract" 2>/dev/null | tr -d '\r')"
  if [ "$existing_code" != "0x" ] && [ "$existing_code" != "0x0" ]; then
    echo "usdt metadata reused in eth.json: $ETH_ARTIFACT_FILE"
    exit 0
  fi

  echo "eth artifact has stale usdt contract address. run: make chain-down-eth && make chain-up-eth" >&2
  exit 1
fi

creation_bytecode="$(forge inspect src/MockUSDT.sol:MockUSDT bytecode --root "$CONTRACT_ROOT" | tr -d '\r')"
constructor_args="$(cast abi-encode "constructor(address,uint256)" "$USDT_MINT_TO" "$USDT_MINT_AMOUNT" | tr -d '\r')"
creation_data="${creation_bytecode}${constructor_args#0x}"

deploy_output="$(cast send --rpc-url "$ETH_RPC_URL" --private-key "$artifact_payer_private_key" --create "$creation_data" | tr -d '\r')"
deploy_tx_hash="$(printf '%s\n' "$deploy_output" | awk '$1=="transactionHash"{print $2; exit}')"
contract_address="$(printf '%s\n' "$deploy_output" | awk '$1=="contractAddress"{print $2; exit}')"

if ! is_hex_hash "$deploy_tx_hash"; then
  echo "failed to capture deployment transaction hash" >&2
  exit 1
fi

if ! is_evm_address "$contract_address" || [ "$contract_address" = "0x0000000000000000000000000000000000000000" ]; then
  echo "failed to resolve deployed contract address from deployment output: $deploy_tx_hash" >&2
  exit 1
fi

minted_balance="$(cast call --rpc-url "$ETH_RPC_URL" "$contract_address" "balanceOf(address)(uint256)" "$USDT_MINT_TO" | tr -d '\r')"

cat >"$ETH_ARTIFACT_FILE" <<JSON
{
  "schema_version": 1,
  "generated_at": "$(utc_now)",
  "network": "local-evm",
  "compose_project": "$artifact_compose_project",
  "warnings": [],
  "rpc_url": "$artifact_rpc_url",
  "chain_id": $actual_chain_id,
  "genesis_block_hash": "$actual_genesis",
  "payer_private_key": "$artifact_payer_private_key",
  "payer_address": "$artifact_payer_address",
  "receiver_xpub": "$artifact_receiver_xpub",
  "usdt_contract_address": "$contract_address",
  "usdt_token_decimals": $USDT_TOKEN_DECIMALS,
  "usdt_minted_to": "$USDT_MINT_TO",
  "usdt_minted_amount": "$USDT_MINT_AMOUNT",
  "usdt_minted_balance": "$minted_balance",
  "usdt_deploy_tx_hash": "$deploy_tx_hash"
}
JSON

echo "usdt deployed: $contract_address"
echo "eth artifact updated with usdt metadata: $ETH_ARTIFACT_FILE"
