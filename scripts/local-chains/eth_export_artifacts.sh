#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd docker
require_cmd jq

ensure_artifact_dir

ETH_ARTIFACT_FILE="${ETH_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/eth.json}"
ETH_PROJECT_NAME="$LOCAL_ETH_PROJECT"
PAYER_PRIVATE_KEY="${PAYER_PRIVATE_KEY:-$ANVIL_DEFAULT_PRIVATE_KEY}"
PAYER_ADDRESS="${PAYER_ADDRESS:-$ANVIL_DEFAULT_ADDRESS}"
ETH_EXPORT_TIMEOUT_SECONDS="${ETH_EXPORT_TIMEOUT_SECONDS:-90}"
ETH_INTERNAL_RPC_URL="${ETH_INTERNAL_RPC_URL:-http://127.0.0.1:8545}"
ETH_RECEIVER_XPUB="${ETH_RECEIVER_XPUB:-${SERVICE_KS_ETH_SEPOLIA:-xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj}}"
USDT_CONTRACT_ADDRESS=""
USDT_TOKEN_DECIMALS=""
USDT_MINTED_TO=""
USDT_MINTED_AMOUNT=""
USDT_MINTED_BALANCE=""
USDT_DEPLOY_TX_HASH=""

if [ -z "$ETH_RECEIVER_XPUB" ]; then
  echo "missing ETH receiver xpub (set ETH_RECEIVER_XPUB or SERVICE_KS_ETH_SEPOLIA)" >&2
  exit 1
fi

eth_dc() {
  dc "$ETH_COMPOSE_FILE" "$LOCAL_ETH_PROJECT" "$@"
}

deadline=$((SECONDS + ETH_EXPORT_TIMEOUT_SECONDS))
chain_id_dec=""
while (( SECONDS < deadline )); do
  if chain_id_dec="$(eth_dc exec -T eth-node cast chain-id --rpc-url "$ETH_INTERNAL_RPC_URL" 2>/dev/null)"; then
    if [ -n "$chain_id_dec" ] && [ "$chain_id_dec" != "null" ]; then
      break
    fi
  fi
  sleep 2
done

if [ -z "$chain_id_dec" ] || [ "$chain_id_dec" = "null" ]; then
  echo "failed to query eth chain id via eth-node container" >&2
  exit 1
fi

if [ "$chain_id_dec" != "$ETH_EXPECTED_CHAIN_ID" ]; then
  echo "unexpected chain id: got $chain_id_dec expected $ETH_EXPECTED_CHAIN_ID" >&2
  exit 1
fi

genesis_block_hash="$(eth_dc exec -T eth-node cast block 0 --rpc-url "$ETH_INTERNAL_RPC_URL" --json | jq -r '.hash')"
if [ -z "$genesis_block_hash" ] || [ "$genesis_block_hash" = "null" ]; then
  echo "failed to read genesis block hash" >&2
  exit 1
fi

if [ -f "$ETH_ARTIFACT_FILE" ]; then
  existing_chain_id="$(jq -r '.chain_id // empty' "$ETH_ARTIFACT_FILE")"
  existing_genesis_hash="$(jq -r '.genesis_block_hash // empty' "$ETH_ARTIFACT_FILE")"

  if [ "$existing_chain_id" = "$chain_id_dec" ] && [ "$existing_genesis_hash" = "$genesis_block_hash" ]; then
    USDT_CONTRACT_ADDRESS="$(jq -r '.usdt_contract_address // empty' "$ETH_ARTIFACT_FILE")"
    USDT_TOKEN_DECIMALS="$(jq -r '.usdt_token_decimals // empty' "$ETH_ARTIFACT_FILE")"
    USDT_MINTED_TO="$(jq -r '.usdt_minted_to // empty' "$ETH_ARTIFACT_FILE")"
    USDT_MINTED_AMOUNT="$(jq -r '.usdt_minted_amount // empty' "$ETH_ARTIFACT_FILE")"
    USDT_MINTED_BALANCE="$(jq -r '.usdt_minted_balance // empty' "$ETH_ARTIFACT_FILE")"
    USDT_DEPLOY_TX_HASH="$(jq -r '.usdt_deploy_tx_hash // empty' "$ETH_ARTIFACT_FILE")"
  fi
fi

jq -n \
  --arg generated_at "$(utc_now)" \
  --arg compose_project "$ETH_PROJECT_NAME" \
  --arg rpc_url "$ETH_RPC_URL" \
  --argjson chain_id "$chain_id_dec" \
  --arg genesis_block_hash "$genesis_block_hash" \
  --arg payer_private_key "$PAYER_PRIVATE_KEY" \
  --arg payer_address "$PAYER_ADDRESS" \
  --arg receiver_xpub "$ETH_RECEIVER_XPUB" \
  --arg usdt_contract_address "$USDT_CONTRACT_ADDRESS" \
  --arg usdt_token_decimals "$USDT_TOKEN_DECIMALS" \
  --arg usdt_minted_to "$USDT_MINTED_TO" \
  --arg usdt_minted_amount "$USDT_MINTED_AMOUNT" \
  --arg usdt_minted_balance "$USDT_MINTED_BALANCE" \
  --arg usdt_deploy_tx_hash "$USDT_DEPLOY_TX_HASH" \
  '{
    schema_version: 1,
    generated_at: $generated_at,
    network: "local-evm",
    compose_project: $compose_project,
    warnings: [],
    rpc_url: $rpc_url,
    chain_id: $chain_id,
    genesis_block_hash: $genesis_block_hash,
    payer_private_key: $payer_private_key,
    payer_address: $payer_address,
    receiver_xpub: $receiver_xpub,
    usdt_contract_address: (if $usdt_contract_address == "" then null else $usdt_contract_address end),
    usdt_token_decimals: (if $usdt_token_decimals == "" then null else ($usdt_token_decimals | tonumber) end),
    usdt_minted_to: (if $usdt_minted_to == "" then null else $usdt_minted_to end),
    usdt_minted_amount: (if $usdt_minted_amount == "" then null else $usdt_minted_amount end),
    usdt_minted_balance: (if $usdt_minted_balance == "" then null else $usdt_minted_balance end),
    usdt_deploy_tx_hash: (if $usdt_deploy_tx_hash == "" then null else $usdt_deploy_tx_hash end)
  }' >"$ETH_ARTIFACT_FILE"

echo "eth artifact exported: $ETH_ARTIFACT_FILE"
