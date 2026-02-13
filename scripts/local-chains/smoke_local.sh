#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd curl
require_cmd jq
require_cmd docker

ensure_artifact_dir

BTC_ARTIFACT_FILE="${BTC_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/btc.json}"
SMOKE_ARTIFACT_FILE="${SMOKE_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/smoke-local.json}"
BTC_SMOKE_AMOUNT="${BTC_SMOKE_AMOUNT:-0.001}"
BTC_DERIVATION_INDEX="${BTC_DERIVATION_INDEX:-1}"

if [ ! -f "$BTC_ARTIFACT_FILE" ]; then
  echo "missing BTC artifact: $BTC_ARTIFACT_FILE (run make chain-up-btc first)" >&2
  exit 1
fi

btc_dc() {
  dc "$BTC_COMPOSE_FILE" "$LOCAL_BTC_PROJECT" "$@"
}

btc_cli() {
  btc_dc exec -T btc-node bitcoin-cli -regtest -rpcuser="$BTC_RPC_USER" -rpcpassword="$BTC_RPC_PASSWORD" "$@"
}

health_code="$(curl -s -o /tmp/chaintx-healthz.json -w "%{http_code}" "$SERVICE_HEALTH_URL" || true)"
if [ "$health_code" != "200" ]; then
  echo "service health check failed at $SERVICE_HEALTH_URL (status=$health_code)" >&2
  exit 1
fi

payer_wallet="$(jq -r '.payer_wallet' "$BTC_ARTIFACT_FILE")"
receiver_descriptor="$(jq -r '.receiver_descriptor' "$BTC_ARTIFACT_FILE")"

if [ -z "$payer_wallet" ] || [ "$payer_wallet" = "null" ]; then
  echo "invalid BTC artifact: missing payer_wallet" >&2
  exit 1
fi
if [ -z "$receiver_descriptor" ] || [ "$receiver_descriptor" = "null" ]; then
  echo "invalid BTC artifact: missing receiver_descriptor" >&2
  exit 1
fi

receiver_address="$(btc_cli deriveaddresses "$receiver_descriptor" "[$BTC_DERIVATION_INDEX,$BTC_DERIVATION_INDEX]" | jq -r '.[0]')"
if [ -z "$receiver_address" ] || [ "$receiver_address" = "null" ]; then
  echo "failed to derive receiver address from descriptor" >&2
  exit 1
fi

balance_before="$(btc_cli -rpcwallet="$payer_wallet" getbalance)"
txid="$(btc_cli -rpcwallet="$payer_wallet" sendtoaddress "$receiver_address" "$BTC_SMOKE_AMOUNT")"
mine_addr="$(btc_cli -rpcwallet="$payer_wallet" getnewaddress "" bech32)"
btc_cli -rpcwallet="$payer_wallet" generatetoaddress 1 "$mine_addr" >/dev/null
confirmations="$(btc_cli -rpcwallet="$payer_wallet" gettransaction "$txid" | jq -r '.confirmations')"
balance_after="$(btc_cli -rpcwallet="$payer_wallet" getbalance)"

jq -n \
  --arg generated_at "$(utc_now)" \
  --arg smoke_type "default" \
  --arg health_url "$SERVICE_HEALTH_URL" \
  --arg txid "$txid" \
  --arg receiver_address "$receiver_address" \
  --arg amount_btc "$BTC_SMOKE_AMOUNT" \
  --arg balance_before "$balance_before" \
  --arg balance_after "$balance_after" \
  --arg confirmations "$confirmations" \
  '{
    schema_version: 1,
    generated_at: $generated_at,
    smoke_type: $smoke_type,
    status: "pass",
    checks: {
      service_health_url: $health_url,
      btc_txid: $txid,
      btc_receiver_address: $receiver_address,
      btc_amount: $amount_btc,
      btc_balance_before: $balance_before,
      btc_balance_after: $balance_after,
      btc_confirmations: ($confirmations | tonumber)
    }
  }' >"$SMOKE_ARTIFACT_FILE"

echo "default smoke passed: $SMOKE_ARTIFACT_FILE"
