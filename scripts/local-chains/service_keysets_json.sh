#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd jq

BTC_ARTIFACT_FILE="${BTC_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/btc.json}"
ETH_ARTIFACT_FILE="${ETH_ARTIFACT_FILE:-$LOCAL_CHAIN_ARTIFACT_DIR/eth.json}"

SERVICE_KS_BTC_REGTEST="${SERVICE_KS_BTC_REGTEST:-}"
SERVICE_KS_BTC_TESTNET="${SERVICE_KS_BTC_TESTNET:-}"
SERVICE_KS_ETH_SEPOLIA="${SERVICE_KS_ETH_SEPOLIA:-}"
SERVICE_KS_ETH_LOCAL="${SERVICE_KS_ETH_LOCAL:-$SERVICE_KS_ETH_SEPOLIA}"

resolve_btc_regtest_keyset() {
  if [ -f "$BTC_ARTIFACT_FILE" ]; then
    local artifact_keyset
    artifact_keyset="$(jq -r '.receiver_xpub // empty' "$BTC_ARTIFACT_FILE")"
    if [ -z "$artifact_keyset" ]; then
      echo "invalid btc artifact: missing receiver_xpub in $BTC_ARTIFACT_FILE" >&2
      exit 1
    fi
    printf '%s' "$artifact_keyset"
    return 0
  fi

  printf '%s' "$SERVICE_KS_BTC_REGTEST"
}

resolve_eth_local_keyset() {
  if [ -f "$ETH_ARTIFACT_FILE" ]; then
    local artifact_keyset
    artifact_keyset="$(jq -r '.receiver_xpub // empty' "$ETH_ARTIFACT_FILE")"
    if [ -z "$artifact_keyset" ]; then
      echo "invalid eth artifact: missing receiver_xpub in $ETH_ARTIFACT_FILE" >&2
      exit 1
    fi
    printf '%s' "$artifact_keyset"
    return 0
  fi

  printf '%s' "$SERVICE_KS_ETH_LOCAL"
}

ks_btc_regtest="$(resolve_btc_regtest_keyset)"
ks_btc_testnet="$SERVICE_KS_BTC_TESTNET"
ks_eth_sepolia="$SERVICE_KS_ETH_SEPOLIA"
ks_eth_local="$(resolve_eth_local_keyset)"

if [ -z "$ks_btc_regtest" ]; then
  echo "missing ks_btc_regtest (set SERVICE_KS_BTC_REGTEST or generate $BTC_ARTIFACT_FILE)" >&2
  exit 1
fi

if [ -z "$ks_btc_testnet" ]; then
  ks_btc_testnet="$ks_btc_regtest"
fi

if [ -z "$ks_eth_sepolia" ]; then
  echo "missing ks_eth_sepolia (set SERVICE_KS_ETH_SEPOLIA or generate $ETH_ARTIFACT_FILE)" >&2
  exit 1
fi

if [ -z "$ks_eth_local" ]; then
  echo "missing ks_eth_local (set SERVICE_KS_ETH_LOCAL or generate $ETH_ARTIFACT_FILE)" >&2
  exit 1
fi

jq -cn \
  --arg reg "$ks_btc_regtest" \
  --arg test "$ks_btc_testnet" \
  --arg eth "$ks_eth_sepolia" \
  --arg eth_local "$ks_eth_local" \
  '{
    ks_btc_regtest: $reg,
    ks_btc_testnet: $test,
    ks_eth_sepolia: $eth,
    ks_eth_local: $eth_local
  }'
