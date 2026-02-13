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

jq -n \
  --arg generated_at "$(utc_now)" \
  --arg compose_project "$ETH_PROJECT_NAME" \
  --arg rpc_url "$ETH_RPC_URL" \
  --argjson chain_id "$chain_id_dec" \
  --arg genesis_block_hash "$genesis_block_hash" \
  --arg payer_private_key "$PAYER_PRIVATE_KEY" \
  --arg payer_address "$PAYER_ADDRESS" \
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
    payer_address: $payer_address
  }' >"$ETH_ARTIFACT_FILE"

echo "eth artifact exported: $ETH_ARTIFACT_FILE"
