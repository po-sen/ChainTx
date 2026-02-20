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
SERVICE_KS_BTC_REGTEST_INDEX0_ADDRESS="${SERVICE_KS_BTC_REGTEST_INDEX0_ADDRESS:-bcrt1q7xfwy8t0z9xar2klctmdgm96kxvg9k8jn30qfg}"
SERVICE_KS_BTC_TESTNET_INDEX0_ADDRESS="${SERVICE_KS_BTC_TESTNET_INDEX0_ADDRESS:-tb1q7xfwy8t0z9xar2klctmdgm96kxvg9k8j3ckd7p}"
SERVICE_KS_ETH_SEPOLIA_INDEX0_ADDRESS="${SERVICE_KS_ETH_SEPOLIA_INDEX0_ADDRESS:-0x61ed32e69db70c5abab0522d80e8f5db215965de}"
SERVICE_KS_ETH_LOCAL_INDEX0_ADDRESS="${SERVICE_KS_ETH_LOCAL_INDEX0_ADDRESS:-$SERVICE_KS_ETH_SEPOLIA_INDEX0_ADDRESS}"

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

resolve_btc_regtest_index0_address() {
  if [ -f "$BTC_ARTIFACT_FILE" ]; then
    local artifact_address
    artifact_address="$(jq -r '.receiver_address_index0 // empty' "$BTC_ARTIFACT_FILE")"
    if [ -n "$artifact_address" ]; then
      printf '%s' "$artifact_address"
      return 0
    fi
  fi

  printf '%s' "$SERVICE_KS_BTC_REGTEST_INDEX0_ADDRESS"
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

resolve_eth_local_index0_address() {
  if [ -f "$ETH_ARTIFACT_FILE" ]; then
    local artifact_address
    artifact_address="$(jq -r '.receiver_address_index0 // empty' "$ETH_ARTIFACT_FILE")"
    if [ -n "$artifact_address" ]; then
      printf '%s' "$artifact_address"
      return 0
    fi
  fi

  printf '%s' "$SERVICE_KS_ETH_LOCAL_INDEX0_ADDRESS"
}

ks_btc_regtest="$(resolve_btc_regtest_keyset)"
ks_btc_testnet="$SERVICE_KS_BTC_TESTNET"
ks_eth_sepolia="$SERVICE_KS_ETH_SEPOLIA"
ks_eth_local="$(resolve_eth_local_keyset)"
addr_btc_regtest="$(resolve_btc_regtest_index0_address)"
addr_btc_testnet="$SERVICE_KS_BTC_TESTNET_INDEX0_ADDRESS"
addr_eth_sepolia="$SERVICE_KS_ETH_SEPOLIA_INDEX0_ADDRESS"
addr_eth_local="$(resolve_eth_local_index0_address)"

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

if [ -z "$addr_btc_regtest" ]; then
  echo "missing ks_btc_regtest index0 address (set SERVICE_KS_BTC_REGTEST_INDEX0_ADDRESS or add receiver_address_index0 in $BTC_ARTIFACT_FILE)" >&2
  exit 1
fi

if [ -z "$addr_btc_testnet" ]; then
  echo "missing ks_btc_testnet index0 address (set SERVICE_KS_BTC_TESTNET_INDEX0_ADDRESS)" >&2
  exit 1
fi

if [ -z "$addr_eth_sepolia" ]; then
  echo "missing ks_eth_sepolia index0 address (set SERVICE_KS_ETH_SEPOLIA_INDEX0_ADDRESS)" >&2
  exit 1
fi

if [ -z "$addr_eth_local" ]; then
  echo "missing ks_eth_local index0 address (set SERVICE_KS_ETH_LOCAL_INDEX0_ADDRESS or add receiver_address_index0 in $ETH_ARTIFACT_FILE)" >&2
  exit 1
fi

jq -cn \
  --arg reg "$ks_btc_regtest" \
  --arg test "$ks_btc_testnet" \
  --arg eth "$ks_eth_sepolia" \
  --arg eth_local "$ks_eth_local" \
  --arg reg_addr "$addr_btc_regtest" \
  --arg test_addr "$addr_btc_testnet" \
  --arg eth_addr "$addr_eth_sepolia" \
  --arg eth_local_addr "$addr_eth_local" \
  '{
    bitcoin: {
      regtest: {
        keyset_id: "ks_btc_regtest",
        extended_public_key: $reg,
        expected_index0_address: $reg_addr
      },
      testnet: {
        keyset_id: "ks_btc_testnet",
        extended_public_key: $test,
        expected_index0_address: $test_addr
      }
    },
    ethereum: {
      sepolia: {
        keyset_id: "ks_eth_sepolia",
        extended_public_key: $eth,
        expected_index0_address: $eth_addr
      },
      local: {
        keyset_id: "ks_eth_local",
        extended_public_key: $eth_local,
        expected_index0_address: $eth_local_addr
      }
    }
  }'
