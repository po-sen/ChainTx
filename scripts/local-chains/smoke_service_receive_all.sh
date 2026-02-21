#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd jq
require_cmd curl
require_cmd docker

ensure_artifact_dir

SERVICE_BASE_URL="${SERVICE_BASE_URL:-http://127.0.0.1:${SERVICE_APP_PORT:-8080}}"
SERVICE_RECEIVE_PROOF_FILE="${SERVICE_RECEIVE_PROOF_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/service-receive-local-all.json}"
ETH_ARTIFACT_FILE="${ETH_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/eth.json}"

BTC_PAYER_WALLET="${BTC_PAYER_WALLET:-chaintx-btc-payer}"
BTC_SEND_AMOUNT="${BTC_SEND_AMOUNT:-0.0005}"
BTC_EXPECTED_MINOR="${BTC_EXPECTED_MINOR:-50000}"

ETH_SENDER_PRIVATE_KEY="${ETH_SENDER_PRIVATE_KEY:-$ANVIL_DEFAULT_PRIVATE_KEY}"
ETH_SEND_AMOUNT_WEI="${ETH_SEND_AMOUNT_WEI:-1000000000000000}"

USDT_SEND_AMOUNT_MINOR="${USDT_SEND_AMOUNT_MINOR:-1000000}"
PAYMENT_REQUEST_WEBHOOK_URL="${PAYMENT_REQUEST_WEBHOOK_URL:-http://localhost:18080/chaintx/webhook}"

ERC20_TRANSFER_TOPIC0="0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

if [ ! -f "$ETH_ARTIFACT_FILE" ]; then
  echo "missing ETH artifact: $ETH_ARTIFACT_FILE (run make chain-up-eth first)" >&2
  exit 1
fi

usdt_contract="$(jq -r '.usdt_contract_address // empty' "$ETH_ARTIFACT_FILE")"
if [ -z "$usdt_contract" ] || [ "$usdt_contract" = "null" ]; then
  echo "missing usdt_contract_address in $ETH_ARTIFACT_FILE (run make chain-up-eth first)" >&2
  exit 1
fi

btc_dc() {
  dc "$BTC_COMPOSE_FILE" "$LOCAL_BTC_PROJECT" "$@"
}

btc_cli() {
  btc_dc exec -T btc-node bitcoin-cli -regtest -rpcuser="$BTC_RPC_USER" -rpcpassword="$BTC_RPC_PASSWORD" "$@"
}

eth_dc() {
  dc "$ETH_COMPOSE_FILE" "$LOCAL_ETH_PROJECT" "$@"
}

eth_cast() {
  eth_dc exec -T eth-node cast "$@"
}

to_dec() {
  local raw="$1"
  local converted
  if [ -z "$raw" ] || [ "$raw" = "null" ]; then
    echo "0"
    return 0
  fi
  if [[ "$raw" =~ ^0x ]]; then
    converted="$(eth_cast to-dec "$raw" | tr -d '\r')"
    printf '%s\n' "$converted" | awk '{print $1}'
    return 0
  fi
  printf '%s\n' "$raw" | awk '{print $1}'
}

wait_for_service() {
  local retries="${SERVICE_HEALTH_RETRIES:-60}"
  local sleep_seconds="${SERVICE_HEALTH_SLEEP_SECONDS:-1}"
  local i
  local health

  for ((i = 1; i <= retries; i += 1)); do
    if health="$(curl -fsS "$SERVICE_BASE_URL/healthz" 2>/dev/null)"; then
      if [ "$(printf '%s' "$health" | jq -r '.status // empty')" = "ok" ]; then
        return 0
      fi
    fi
    sleep "$sleep_seconds"
  done

  echo "service health check failed: $SERVICE_BASE_URL/healthz" >&2
  exit 1
}

create_payment_request() {
  local chain="$1"
  local network="$2"
  local asset="$3"
  local expected_minor="$4"
  local idem_key

  idem_key="local-receive-$(date +%s%N)-${asset,,}"

  curl -fsS -X POST "$SERVICE_BASE_URL/v1/payment-requests" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: $idem_key" \
    -d "$(
      jq -cn \
        --arg chain "$chain" \
        --arg network "$network" \
        --arg asset "$asset" \
        --arg webhook_url "$PAYMENT_REQUEST_WEBHOOK_URL" \
        --arg expected "$expected_minor" \
        '{
          chain: $chain,
          network: $network,
          asset: $asset,
          webhook_url: $webhook_url,
          expected_amount_minor: $expected
        }'
    )"
}

eth_chain_id="$(eth_cast chain-id --rpc-url "$ETH_RPC_URL" | tr -d '\r')"
if [ "$eth_chain_id" != "$ETH_EXPECTED_CHAIN_ID" ]; then
  echo "unexpected ETH chain id: got $eth_chain_id expected $ETH_EXPECTED_CHAIN_ID" >&2
  exit 1
fi

btc_cli getblockchaininfo >/dev/null
wait_for_service

# BTC proof
btc_pr="$(create_payment_request bitcoin regtest BTC "$BTC_EXPECTED_MINOR")"
btc_pr_id="$(printf '%s' "$btc_pr" | jq -r '.id // empty')"
btc_addr="$(printf '%s' "$btc_pr" | jq -r '.payment_instructions.address // empty')"
if [ -z "$btc_pr_id" ] || [ -z "$btc_addr" ]; then
  echo "invalid BTC payment request response: missing id/address" >&2
  exit 1
fi

btc_txid="$(btc_cli -rpcwallet="$BTC_PAYER_WALLET" sendtoaddress "$btc_addr" "$BTC_SEND_AMOUNT" | tr -d '\r')"
btc_mine_addr="$(btc_cli -rpcwallet="$BTC_PAYER_WALLET" getnewaddress "" bech32 | tr -d '\r')"
btc_cli -rpcwallet="$BTC_PAYER_WALLET" generatetoaddress 1 "$btc_mine_addr" >/dev/null

btc_tx_verbose="$(btc_cli getrawtransaction "$btc_txid" true)"
btc_vout_index="$(printf '%s' "$btc_tx_verbose" | jq --arg addr "$btc_addr" -r '[.vout[] | select((.scriptPubKey.address // empty) == $addr or ((.scriptPubKey.addresses // []) | index($addr) != null))][0].n // empty')"
if [ -z "$btc_vout_index" ]; then
  echo "btc tx does not contain requested recipient output: $btc_addr" >&2
  exit 1
fi

btc_txout="$(btc_cli gettxout "$btc_txid" "$btc_vout_index" true)"
if [ "$btc_txout" = "null" ]; then
  echo "btc txout not found for txid=$btc_txid vout=$btc_vout_index" >&2
  exit 1
fi

btc_confirmations="$(printf '%s' "$btc_txout" | jq -r '.confirmations // 0')"
if [ "$btc_confirmations" -lt 1 ]; then
  echo "btc txout confirmations < 1 (got $btc_confirmations)" >&2
  exit 1
fi

btc_txout_addr="$(printf '%s' "$btc_txout" | jq -r '.scriptPubKey.address // empty')"
if [ "$btc_txout_addr" != "$btc_addr" ]; then
  echo "btc txout address mismatch: got $btc_txout_addr expected $btc_addr" >&2
  exit 1
fi

btc_pr_get="$(curl -fsS "$SERVICE_BASE_URL/v1/payment-requests/$btc_pr_id")"
if [ "$(printf '%s' "$btc_pr_get" | jq -r '.payment_instructions.address // empty')" != "$btc_addr" ]; then
  echo "btc payment request fetch mismatch: address changed for $btc_pr_id" >&2
  exit 1
fi

# ETH proof
eth_pr="$(create_payment_request ethereum local ETH "$ETH_SEND_AMOUNT_WEI")"
eth_pr_id="$(printf '%s' "$eth_pr" | jq -r '.id // empty')"
eth_addr="$(printf '%s' "$eth_pr" | jq -r '.payment_instructions.address // empty')"
if [ -z "$eth_pr_id" ] || [ -z "$eth_addr" ]; then
  echo "invalid ETH payment request response: missing id/address" >&2
  exit 1
fi

eth_balance_before_raw="$(eth_cast balance --rpc-url "$ETH_RPC_URL" "$eth_addr" | tr -d '\r')"
eth_balance_before="$(to_dec "$eth_balance_before_raw")"

eth_tx="$(eth_cast send --rpc-url "$ETH_RPC_URL" --private-key "$ETH_SENDER_PRIVATE_KEY" "$eth_addr" --value "$ETH_SEND_AMOUNT_WEI" --json | jq -r '.transactionHash')"
eth_receipt="$(eth_cast receipt --rpc-url "$ETH_RPC_URL" "$eth_tx" --json)"

eth_receipt_status_raw="$(printf '%s' "$eth_receipt" | jq -r '.status // empty')"
eth_receipt_status="$(to_dec "$eth_receipt_status_raw")"
if [ "$eth_receipt_status" != "1" ]; then
  echo "eth receipt status is not success: $eth_receipt_status_raw" >&2
  exit 1
fi

eth_balance_after_raw="$(eth_cast balance --rpc-url "$ETH_RPC_URL" "$eth_addr" | tr -d '\r')"
eth_balance_after="$(to_dec "$eth_balance_after_raw")"
eth_balance_delta="$((eth_balance_after - eth_balance_before))"
if [ "$eth_balance_delta" -lt "$ETH_SEND_AMOUNT_WEI" ]; then
  echo "eth recipient balance delta too small: got $eth_balance_delta expected >= $ETH_SEND_AMOUNT_WEI" >&2
  exit 1
fi

eth_pr_get="$(curl -fsS "$SERVICE_BASE_URL/v1/payment-requests/$eth_pr_id")"
if [ "$(printf '%s' "$eth_pr_get" | jq -r '.payment_instructions.address // empty')" != "$eth_addr" ]; then
  echo "eth payment request fetch mismatch: address changed for $eth_pr_id" >&2
  exit 1
fi

# USDT proof
usdt_pr="$(create_payment_request ethereum local USDT "$USDT_SEND_AMOUNT_MINOR")"
usdt_pr_id="$(printf '%s' "$usdt_pr" | jq -r '.id // empty')"
usdt_addr="$(printf '%s' "$usdt_pr" | jq -r '.payment_instructions.address // empty')"
if [ -z "$usdt_pr_id" ] || [ -z "$usdt_addr" ]; then
  echo "invalid USDT payment request response: missing id/address" >&2
  exit 1
fi

usdt_balance_before_raw="$(eth_cast call --rpc-url "$ETH_RPC_URL" "$usdt_contract" "balanceOf(address)(uint256)" "$usdt_addr" | tr -d '\r')"
usdt_balance_before="$(to_dec "$usdt_balance_before_raw")"

usdt_tx="$(eth_cast send --rpc-url "$ETH_RPC_URL" --private-key "$ETH_SENDER_PRIVATE_KEY" "$usdt_contract" "transfer(address,uint256)" "$usdt_addr" "$USDT_SEND_AMOUNT_MINOR" --json | jq -r '.transactionHash')"
usdt_receipt="$(eth_cast receipt --rpc-url "$ETH_RPC_URL" "$usdt_tx" --json)"

usdt_receipt_status_raw="$(printf '%s' "$usdt_receipt" | jq -r '.status // empty')"
usdt_receipt_status="$(to_dec "$usdt_receipt_status_raw")"
if [ "$usdt_receipt_status" != "1" ]; then
  echo "usdt receipt status is not success: $usdt_receipt_status_raw" >&2
  exit 1
fi

usdt_balance_after_raw="$(eth_cast call --rpc-url "$ETH_RPC_URL" "$usdt_contract" "balanceOf(address)(uint256)" "$usdt_addr" | tr -d '\r')"
usdt_balance_after="$(to_dec "$usdt_balance_after_raw")"
usdt_balance_delta="$((usdt_balance_after - usdt_balance_before))"
if [ "$usdt_balance_delta" -lt "$USDT_SEND_AMOUNT_MINOR" ]; then
  echo "usdt recipient balance delta too small: got $usdt_balance_delta expected >= $USDT_SEND_AMOUNT_MINOR" >&2
  exit 1
fi

usdt_topic_recipient="0x000000000000000000000000$(printf '%040s' "${usdt_addr#0x}" | tr '[:upper:]' '[:lower:]' | tr ' ' '0')"
usdt_transfer_log_count="$(printf '%s' "$usdt_receipt" | jq --arg topic0 "$ERC20_TRANSFER_TOPIC0" --arg recipient "$usdt_topic_recipient" '[.logs[]? | select((.topics[0] // "" | ascii_downcase) == ($topic0 | ascii_downcase) and (.topics[2] // "" | ascii_downcase) == ($recipient | ascii_downcase))] | length')"
if [ "$usdt_transfer_log_count" -lt 1 ]; then
  echo "usdt transfer log not found for recipient: $usdt_addr" >&2
  exit 1
fi

usdt_pr_get="$(curl -fsS "$SERVICE_BASE_URL/v1/payment-requests/$usdt_pr_id")"
if [ "$(printf '%s' "$usdt_pr_get" | jq -r '.payment_instructions.address // empty')" != "$usdt_addr" ]; then
  echo "usdt payment request fetch mismatch: address changed for $usdt_pr_id" >&2
  exit 1
fi

jq -n \
  --arg generated_at "$(utc_now)" \
  --arg status "pass" \
  --arg service_base_url "$SERVICE_BASE_URL" \
  --argjson btc_payment_request "$btc_pr" \
  --argjson btc_payment_request_fetched "$btc_pr_get" \
  --arg btc_txid "$btc_txid" \
  --arg btc_recipient "$btc_addr" \
  --arg btc_send_amount "$BTC_SEND_AMOUNT" \
  --argjson btc_tx_verbose "$btc_tx_verbose" \
  --argjson btc_txout "$btc_txout" \
  --arg btc_vout_index "$btc_vout_index" \
  --argjson eth_payment_request "$eth_pr" \
  --argjson eth_payment_request_fetched "$eth_pr_get" \
  --arg eth_tx "$eth_tx" \
  --arg eth_recipient "$eth_addr" \
  --arg eth_send_amount_wei "$ETH_SEND_AMOUNT_WEI" \
  --argjson eth_receipt "$eth_receipt" \
  --arg eth_balance_before "$eth_balance_before" \
  --arg eth_balance_after "$eth_balance_after" \
  --arg eth_balance_delta "$eth_balance_delta" \
  --argjson usdt_payment_request "$usdt_pr" \
  --argjson usdt_payment_request_fetched "$usdt_pr_get" \
  --arg usdt_tx "$usdt_tx" \
  --arg usdt_recipient "$usdt_addr" \
  --arg usdt_contract "$usdt_contract" \
  --arg usdt_send_amount_minor "$USDT_SEND_AMOUNT_MINOR" \
  --argjson usdt_receipt "$usdt_receipt" \
  --arg usdt_balance_before "$usdt_balance_before" \
  --arg usdt_balance_after "$usdt_balance_after" \
  --arg usdt_balance_delta "$usdt_balance_delta" \
  --arg usdt_transfer_log_count "$usdt_transfer_log_count" \
  '{
    schema_version: 1,
    generated_at: $generated_at,
    status: $status,
    service_base_url: $service_base_url,
    checks: {
      btc: {
        payment_request: $btc_payment_request,
        payment_request_fetched: $btc_payment_request_fetched,
        onchain_proof: {
          recipient_address: $btc_recipient,
          send_amount_btc: $btc_send_amount,
          txid: $btc_txid,
          vout_index: ($btc_vout_index | tonumber),
          tx_verbose: {
            blockhash: $btc_tx_verbose.blockhash,
            blockheight: $btc_tx_verbose.blockheight,
            confirmations: $btc_tx_verbose.confirmations
          },
          utxo: {
            value_btc: $btc_txout.value,
            confirmations: $btc_txout.confirmations,
            script_pub_key: $btc_txout.scriptPubKey
          }
        }
      },
      eth: {
        payment_request: $eth_payment_request,
        payment_request_fetched: $eth_payment_request_fetched,
        onchain_proof: {
          recipient_address: $eth_recipient,
          send_amount_wei: $eth_send_amount_wei,
          tx_hash: $eth_tx,
          receipt: {
            blockHash: $eth_receipt.blockHash,
            blockNumber: $eth_receipt.blockNumber,
            from: $eth_receipt.from,
            to: $eth_receipt.to,
            gasUsed: $eth_receipt.gasUsed,
            status: $eth_receipt.status
          },
          recipient_balance_before_wei: $eth_balance_before,
          recipient_balance_after_wei: $eth_balance_after,
          recipient_balance_delta_wei: $eth_balance_delta
        }
      },
      usdt: {
        payment_request: $usdt_payment_request,
        payment_request_fetched: $usdt_payment_request_fetched,
        onchain_proof: {
          recipient_address: $usdt_recipient,
          contract: $usdt_contract,
          send_amount_minor: $usdt_send_amount_minor,
          tx_hash: $usdt_tx,
          receipt: {
            blockHash: $usdt_receipt.blockHash,
            blockNumber: $usdt_receipt.blockNumber,
            from: $usdt_receipt.from,
            to: $usdt_receipt.to,
            gasUsed: $usdt_receipt.gasUsed,
            status: $usdt_receipt.status
          },
          transfer_log_count_to_recipient: ($usdt_transfer_log_count | tonumber),
          recipient_balance_before_minor: $usdt_balance_before,
          recipient_balance_after_minor: $usdt_balance_after,
          recipient_balance_delta_minor: $usdt_balance_delta
        }
      }
    }
  }' >"$SERVICE_RECEIVE_PROOF_FILE"

echo "service receive smoke passed: $SERVICE_RECEIVE_PROOF_FILE"
