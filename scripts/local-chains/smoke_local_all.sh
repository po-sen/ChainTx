#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd jq
require_cmd docker

"$SCRIPT_DIR/smoke_local.sh"

ETH_ARTIFACT_FILE="${ETH_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/eth.json}"
SMOKE_ALL_ARTIFACT_FILE="${SMOKE_ALL_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/smoke-local-all.json}"

if [ ! -f "$ETH_ARTIFACT_FILE" ]; then
  echo "missing ETH artifact: $ETH_ARTIFACT_FILE (run make chain-up-eth first)" >&2
  exit 1
fi

eth_chain_id="$(jq -r '.chain_id' "$ETH_ARTIFACT_FILE")"
eth_genesis="$(jq -r '.genesis_block_hash' "$ETH_ARTIFACT_FILE")"
eth_rpc_url="$(jq -r '.rpc_url // empty' "$ETH_ARTIFACT_FILE")"
eth_private_key="$(jq -r '.payer_private_key // empty' "$ETH_ARTIFACT_FILE")"
eth_payer_address="$(jq -r '.payer_address // empty' "$ETH_ARTIFACT_FILE")"
eth_recipient_address="${ETH_RECIPIENT_ADDRESS:-$ANVIL_SECOND_ADDRESS}"

usdt_rpc_url="$eth_rpc_url"
usdt_private_key="$eth_private_key"
usdt_payer_address="$eth_payer_address"
usdt_contract="$(jq -r '.usdt_contract_address // empty' "$ETH_ARTIFACT_FILE")"
usdt_recipient_address="${USDT_RECIPIENT_ADDRESS:-$ANVIL_SECOND_ADDRESS}"
usdt_transfer_amount="${USDT_TRANSFER_AMOUNT:-1000000}"

if [ -z "$usdt_rpc_url" ] || [ "$usdt_rpc_url" = "null" ]; then
  usdt_rpc_url="$ETH_RPC_URL"
fi

if [ -z "$eth_private_key" ]; then
  eth_private_key="$ANVIL_DEFAULT_PRIVATE_KEY"
fi

if [ -z "$eth_payer_address" ]; then
  eth_payer_address="$ANVIL_DEFAULT_ADDRESS"
fi

eth_dc() {
  dc "$ETH_COMPOSE_FILE" "$LOCAL_ETH_PROJECT" "$@"
}

eth_cast() {
  eth_dc exec -T eth-node cast "$@"
}

actual_eth_chain_id="$(eth_cast chain-id --rpc-url http://127.0.0.1:8545)"
actual_eth_genesis="$(eth_cast block 0 --rpc-url http://127.0.0.1:8545 --json | jq -r '.hash')"

if [ "$actual_eth_chain_id" != "$ETH_EXPECTED_CHAIN_ID" ]; then
  echo "unexpected chain id from ETH RPC: got $actual_eth_chain_id expected $ETH_EXPECTED_CHAIN_ID" >&2
  exit 1
fi

if [ "$eth_chain_id" != "$actual_eth_chain_id" ] || [ "$eth_genesis" != "$actual_eth_genesis" ]; then
  echo "ETH artifact fingerprint mismatch. rerun make chain-up-eth" >&2
  exit 1
fi

if [ -z "$usdt_contract" ] || [ "$usdt_contract" = "null" ]; then
  echo "missing usdt_contract_address in ETH artifact. run make chain-up-eth first" >&2
  exit 1
fi

if [ -z "$usdt_private_key" ]; then
  usdt_private_key="$ANVIL_DEFAULT_PRIVATE_KEY"
fi
if [ -z "$usdt_payer_address" ]; then
  usdt_payer_address="$ANVIL_DEFAULT_ADDRESS"
fi

usdt_code="$(eth_cast code --rpc-url http://127.0.0.1:8545 "$usdt_contract")"
if [ "$usdt_code" = "0x" ] || [ "$usdt_code" = "0x0" ]; then
  echo "USDT contract code missing on current chain. run make chain-up-eth (or chain-down-eth && chain-up-eth)" >&2
  exit 1
fi

eth_balance_before="$(eth_cast balance --rpc-url http://127.0.0.1:8545 "$eth_recipient_address")"
eth_tx="$(
  eth_cast send --rpc-url http://127.0.0.1:8545 --private-key "$eth_private_key" "$eth_recipient_address" --value 1000000000000000 --json | jq -r '.transactionHash'
)"
eth_balance_after="$(eth_cast balance --rpc-url http://127.0.0.1:8545 "$eth_recipient_address")"

usdt_balance_before="$(
  eth_cast call --rpc-url http://127.0.0.1:8545 "$usdt_contract" "balanceOf(address)(uint256)" "$usdt_recipient_address"
)"
usdt_tx="$(
  eth_cast send --rpc-url http://127.0.0.1:8545 --private-key "$usdt_private_key" "$usdt_contract" "transfer(address,uint256)" "$usdt_recipient_address" "$usdt_transfer_amount" --json | jq -r '.transactionHash'
)"
usdt_balance_after="$(
  eth_cast call --rpc-url http://127.0.0.1:8545 "$usdt_contract" "balanceOf(address)(uint256)" "$usdt_recipient_address"
)"

jq -n \
  --arg generated_at "$(utc_now)" \
  --arg smoke_type "full" \
  --arg eth_chain_id "$actual_eth_chain_id" \
  --arg eth_genesis_block_hash "$actual_eth_genesis" \
  --arg eth_payer_address "$eth_payer_address" \
  --arg eth_recipient_address "$eth_recipient_address" \
  --arg eth_tx "$eth_tx" \
  --arg eth_balance_before "$eth_balance_before" \
  --arg eth_balance_after "$eth_balance_after" \
  --arg usdt_rpc_url "$usdt_rpc_url" \
  --arg usdt_payer_address "$usdt_payer_address" \
  --arg usdt_contract "$usdt_contract" \
  --arg usdt_tx "$usdt_tx" \
  --arg usdt_recipient_address "$usdt_recipient_address" \
  --arg usdt_balance_before "$usdt_balance_before" \
  --arg usdt_balance_after "$usdt_balance_after" \
  --arg usdt_transfer_amount "$usdt_transfer_amount" \
  '{
    schema_version: 1,
    generated_at: $generated_at,
    smoke_type: $smoke_type,
    status: "pass",
    checks: {
      eth_transfer: {
        chain_id: ($eth_chain_id | tonumber),
        genesis_block_hash: $eth_genesis_block_hash,
        payer: $eth_payer_address,
        recipient: $eth_recipient_address,
        tx_hash: $eth_tx,
        recipient_balance_before: $eth_balance_before,
        recipient_balance_after: $eth_balance_after
      },
      usdt_transfer: {
        chain_id: ($eth_chain_id | tonumber),
        genesis_block_hash: $eth_genesis_block_hash,
        rpc_url: $usdt_rpc_url,
        payer: $usdt_payer_address,
        contract: $usdt_contract,
        recipient: $usdt_recipient_address,
        tx_hash: $usdt_tx,
        amount: $usdt_transfer_amount,
        recipient_balance_before: $usdt_balance_before,
        recipient_balance_after: $usdt_balance_after
      }
    }
  }' >"$SMOKE_ALL_ARTIFACT_FILE"

echo "full smoke passed: $SMOKE_ALL_ARTIFACT_FILE"
