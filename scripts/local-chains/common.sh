#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

LOCAL_CHAIN_ARTIFACT_DIR="${LOCAL_CHAIN_ARTIFACT_DIR:-$REPO_ROOT/deployments/local-chains/artifacts}"

LOCAL_BTC_PROJECT="${LOCAL_BTC_PROJECT:-chaintx-local-btc}"
LOCAL_ETH_PROJECT="${LOCAL_ETH_PROJECT:-chaintx-local-eth}"
LOCAL_SERVICE_PROJECT="${LOCAL_SERVICE_PROJECT:-chaintx-local-service}"

BTC_COMPOSE_FILE="${BTC_COMPOSE_FILE:-$REPO_ROOT/deployments/local-chains/docker-compose.btc.yml}"
ETH_COMPOSE_FILE="${ETH_COMPOSE_FILE:-$REPO_ROOT/deployments/local-chains/docker-compose.eth.yml}"
SERVICE_COMPOSE_FILE="${SERVICE_COMPOSE_FILE:-$REPO_ROOT/deployments/service/docker-compose.yml}"

BTC_RPC_USER="${BTC_RPC_USER:-chaintx}"
BTC_RPC_PASSWORD="${BTC_RPC_PASSWORD:-chaintx}"
BTC_RPC_URL="${BTC_RPC_URL:-http://127.0.0.1:${BTC_RPC_PORT:-18443}}"

ETH_RPC_URL="${ETH_RPC_URL:-http://127.0.0.1:${ETH_RPC_PORT:-8545}}"
ETH_EXPECTED_CHAIN_ID="${ETH_EXPECTED_CHAIN_ID:-31337}"

SERVICE_HEALTH_URL="${SERVICE_HEALTH_URL:-http://127.0.0.1:${SERVICE_APP_PORT:-8080}/healthz}"

ANVIL_DEFAULT_PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
ANVIL_DEFAULT_ADDRESS="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
ANVIL_SECOND_ADDRESS="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "missing required command: $cmd" >&2
    exit 1
  fi
}

ensure_artifact_dir() {
  mkdir -p "$LOCAL_CHAIN_ARTIFACT_DIR"
}

dc() {
  local compose_file="$1"
  local project="$2"
  shift 2
  docker compose -f "$compose_file" --project-name "$project" "$@"
}

json_rpc() {
  local rpc_url="$1"
  local method="$2"
  local params="${3:-[]}"

  curl -fsS \
    --connect-timeout 2 \
    --max-time 5 \
    -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"$method\",\"params\":$params}" \
    "$rpc_url"
}

hex_to_dec() {
  local hex="$1"
  hex="${hex#0x}"
  printf '%d' "0x$hex"
}

utc_now() {
  date -u +"%Y-%m-%dT%H:%M:%SZ"
}
