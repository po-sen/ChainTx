#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd jq
require_cmd cast
require_cmd forge

ensure_artifact_dir

USDT_ARTIFACT_FILE="${USDT_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/usdt.json}"
USDT_PROJECT_NAME="$LOCAL_USDT_PROJECT"
CONTRACT_ROOT="${CONTRACT_ROOT:-$REPO_ROOT/scripts/local-chains/contracts}"
USDT_MINT_AMOUNT="${USDT_MINT_AMOUNT:-1000000000000}"
USDT_MINT_TO="${USDT_MINT_TO:-$ANVIL_DEFAULT_ADDRESS}"
PAYER_PRIVATE_KEY="${PAYER_PRIVATE_KEY:-$ANVIL_DEFAULT_PRIVATE_KEY}"
PAYER_ADDRESS="${PAYER_ADDRESS:-$ANVIL_DEFAULT_ADDRESS}"
USDT_TOKEN_DECIMALS="${USDT_TOKEN_DECIMALS:-6}"
USDT_ARTIFACT_RPC_URL="${USDT_ARTIFACT_RPC_URL:-$ETH_RPC_URL}"

actual_chain_id="$(cast chain-id --rpc-url "$ETH_RPC_URL")"
actual_genesis="$(cast block 0 --rpc-url "$ETH_RPC_URL" --json | jq -r '.hash')"

if [ "$actual_chain_id" != "$ETH_EXPECTED_CHAIN_ID" ]; then
  echo "unexpected chain id from ETH RPC: got $actual_chain_id expected $ETH_EXPECTED_CHAIN_ID" >&2
  exit 1
fi

if [ -f "$USDT_ARTIFACT_FILE" ]; then
  existing_chain_id="$(jq -r '.chain_id' "$USDT_ARTIFACT_FILE")"
  existing_genesis="$(jq -r '.genesis_block_hash' "$USDT_ARTIFACT_FILE")"
  existing_contract="$(jq -r '.contract_address' "$USDT_ARTIFACT_FILE")"

  if [ "$existing_chain_id" != "$actual_chain_id" ] || [ "$existing_genesis" != "$actual_genesis" ]; then
    echo "stale usdt artifact fingerprint detected. run: make chain-down-usdt && make chain-up-usdt" >&2
    exit 1
  fi

  if [ -n "$existing_contract" ] && [ "$existing_contract" != "null" ]; then
    existing_code="$(cast code --rpc-url "$ETH_RPC_URL" "$existing_contract")"
    if [ "$existing_code" != "0x" ] && [ "$existing_code" != "0x0" ]; then
      echo "usdt artifact reused: $USDT_ARTIFACT_FILE"
      exit 0
    fi
  fi

  echo "usdt artifact exists but contract code is missing. run: make chain-down-usdt && make chain-up-usdt" >&2
  exit 1
fi

creation_bytecode="$(forge inspect src/MockUSDT.sol:MockUSDT bytecode --root "$CONTRACT_ROOT")"
constructor_args="$(cast abi-encode "constructor(address,uint256)" "$USDT_MINT_TO" "$USDT_MINT_AMOUNT")"
creation_data="${creation_bytecode}${constructor_args#0x}"

deploy_tx_hash="$(
  cast send --rpc-url "$ETH_RPC_URL" --private-key "$PAYER_PRIVATE_KEY" --create "$creation_data" --json \
    | jq -r '.transactionHash'
)"
if [ -z "$deploy_tx_hash" ] || [ "$deploy_tx_hash" = "null" ]; then
  echo "failed to capture deployment transaction hash" >&2
  exit 1
fi

contract_address=""
for _ in $(seq 1 45); do
  contract_address="$(cast receipt --rpc-url "$ETH_RPC_URL" "$deploy_tx_hash" --json | jq -r '.contractAddress // empty')"
  if [ -n "$contract_address" ] && [ "$contract_address" != "null" ] && [ "$contract_address" != "0x0000000000000000000000000000000000000000" ]; then
    break
  fi
  sleep 1
done

if [ -z "$contract_address" ] || [ "$contract_address" = "null" ]; then
  echo "failed to resolve deployed contract address from tx receipt: $deploy_tx_hash" >&2
  exit 1
fi

minted_balance="$(cast call --rpc-url "$ETH_RPC_URL" "$contract_address" "balanceOf(address)(uint256)" "$USDT_MINT_TO")"

jq -n \
  --arg generated_at "$(utc_now)" \
  --arg compose_project "$USDT_PROJECT_NAME" \
  --arg rpc_url "$USDT_ARTIFACT_RPC_URL" \
  --argjson chain_id "$actual_chain_id" \
  --arg genesis_block_hash "$actual_genesis" \
  --arg contract_address "$contract_address" \
  --argjson token_decimals "$USDT_TOKEN_DECIMALS" \
  --arg minted_to "$USDT_MINT_TO" \
  --arg minted_amount "$USDT_MINT_AMOUNT" \
  --arg minted_balance "$minted_balance" \
  --arg payer_private_key "$PAYER_PRIVATE_KEY" \
  --arg payer_address "$PAYER_ADDRESS" \
  --arg deploy_tx_hash "$deploy_tx_hash" \
  '{
    schema_version: 1,
    generated_at: $generated_at,
    network: "local-evm",
    compose_project: $compose_project,
    warnings: [],
    rpc_url: $rpc_url,
    chain_id: $chain_id,
    genesis_block_hash: $genesis_block_hash,
    contract_address: $contract_address,
    token_decimals: $token_decimals,
    minted_to: $minted_to,
    minted_amount: $minted_amount,
    minted_balance: $minted_balance,
    payer_private_key: $payer_private_key,
    payer_address: $payer_address,
    deploy_tx_hash: $deploy_tx_hash
  }' >"$USDT_ARTIFACT_FILE"

echo "usdt deployed: $contract_address"
echo "usdt artifact exported: $USDT_ARTIFACT_FILE"
