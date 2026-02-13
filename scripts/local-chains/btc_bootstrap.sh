#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd docker
require_cmd jq
require_cmd grep

ensure_artifact_dir

BTC_PAYER_WALLET="${BTC_PAYER_WALLET:-chaintx-btc-payer}"
BTC_RECEIVER_WALLET="${BTC_RECEIVER_WALLET:-chaintx-btc-receiver}"
BTC_MIN_BALANCE="${BTC_MIN_BALANCE:-1.0}"
BTC_BOOTSTRAP_TIMEOUT_SECONDS="${BTC_BOOTSTRAP_TIMEOUT_SECONDS:-120}"
BTC_ARTIFACT_FILE="${BTC_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/btc.json}"
BTC_PROJECT_NAME="$LOCAL_BTC_PROJECT"

btc_dc() {
  dc "$BTC_COMPOSE_FILE" "$BTC_PROJECT_NAME" "$@"
}

btc_cli() {
  btc_dc exec -T btc-node bitcoin-cli -regtest -rpcuser="$BTC_RPC_USER" -rpcpassword="$BTC_RPC_PASSWORD" "$@"
}

wait_for_btc() {
  local deadline=$((SECONDS + BTC_BOOTSTRAP_TIMEOUT_SECONDS))
  while (( SECONDS < deadline )); do
    if info="$(btc_cli getblockchaininfo 2>/dev/null)"; then
      local chain
      local ibd
      chain="$(printf '%s' "$info" | jq -r '.chain')"
      ibd="$(printf '%s' "$info" | jq -r '.initialblockdownload')"
      # On fresh regtest nodes, IBD may remain true at height 0 until blocks are mined.
      if [ "$ibd" = "false" ] || [ "$chain" = "regtest" ]; then
        return 0
      fi
    fi
    sleep 2
  done

  echo "timed out waiting for BTC node readiness" >&2
  return 1
}

ensure_wallet_loaded() {
  local wallet_name="$1"

  if btc_cli -rpcwallet="$wallet_name" getwalletinfo >/dev/null 2>&1; then
    return 0
  fi

  local create_out
  if ! create_out="$(btc_cli -named createwallet wallet_name="$wallet_name" descriptors=true load_on_startup=true 2>&1)"; then
    if printf '%s' "$create_out" | grep -Eq "already exists|Database already exists"; then
      btc_cli loadwallet "$wallet_name" >/dev/null 2>&1 || true
    else
      echo "$create_out" >&2
      return 1
    fi
  fi

  if ! btc_cli -rpcwallet="$wallet_name" getwalletinfo >/dev/null 2>&1; then
    btc_cli loadwallet "$wallet_name" >/dev/null
  fi
}

wait_for_btc
ensure_wallet_loaded "$BTC_PAYER_WALLET"
ensure_wallet_loaded "$BTC_RECEIVER_WALLET"

payer_balance="$(btc_cli -rpcwallet="$BTC_PAYER_WALLET" getbalance)"
if awk -v bal="$payer_balance" -v min="$BTC_MIN_BALANCE" 'BEGIN { exit !(bal < min) }'; then
  mine_addr="$(btc_cli -rpcwallet="$BTC_PAYER_WALLET" getnewaddress "" bech32)"
  btc_cli -rpcwallet="$BTC_PAYER_WALLET" generatetoaddress 101 "$mine_addr" >/dev/null
fi

payer_balance="$(btc_cli -rpcwallet="$BTC_PAYER_WALLET" getbalance)"
payer_address="$(btc_cli -rpcwallet="$BTC_PAYER_WALLET" getnewaddress "" bech32)"
receiver_descriptor="$(
  btc_cli -rpcwallet="$BTC_RECEIVER_WALLET" listdescriptors false \
    | jq -r '.descriptors[] | select((.internal // false) == false) | .desc' \
    | grep '^wpkh(' \
    | head -n 1 || true
)"

if [ -z "$receiver_descriptor" ] || [ "$receiver_descriptor" = "null" ]; then
  echo "failed to resolve receiver descriptor (wpkh external)" >&2
  exit 1
fi

receiver_xpub="$(printf '%s' "$receiver_descriptor" | grep -Eo '(xpub|tpub|vpub)[1-9A-HJ-NP-Za-km-z]+' | head -n 1 || true)"
if [ -z "$receiver_xpub" ]; then
  echo "failed to extract receiver xpub/tpub/vpub from descriptor" >&2
  exit 1
fi

receiver_address="$(btc_cli -rpcwallet="$BTC_RECEIVER_WALLET" deriveaddresses "$receiver_descriptor" "[0,0]" | jq -r '.[0]')"
block_height="$(btc_cli getblockcount)"

jq -n \
  --arg generated_at "$(utc_now)" \
  --arg compose_project "$BTC_PROJECT_NAME" \
  --arg rpc_url "$BTC_RPC_URL" \
  --arg payer_wallet "$BTC_PAYER_WALLET" \
  --arg payer_address "$payer_address" \
  --arg payer_balance "$payer_balance" \
  --arg receiver_wallet "$BTC_RECEIVER_WALLET" \
  --arg receiver_descriptor "$receiver_descriptor" \
  --arg receiver_xpub "$receiver_xpub" \
  --arg receiver_address "$receiver_address" \
  --arg derivation_template "m/84'/1'/0'/0/{index}" \
  --argjson block_height "$block_height" \
  '{
    schema_version: 1,
    generated_at: $generated_at,
    network: "regtest",
    compose_project: $compose_project,
    warnings: [],
    rpc_url: $rpc_url,
    payer_wallet: $payer_wallet,
    payer_address: $payer_address,
    payer_balance: $payer_balance,
    receiver_wallet: $receiver_wallet,
    receiver_descriptor: $receiver_descriptor,
    receiver_xpub: $receiver_xpub,
    receiver_address_index0: $receiver_address,
    derivation_template: $derivation_template,
    block_height: $block_height
  }' >"$BTC_ARTIFACT_FILE"

echo "btc bootstrap completed: $BTC_ARTIFACT_FILE"
