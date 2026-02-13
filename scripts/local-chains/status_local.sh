#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd docker
require_cmd jq

print_title() {
  printf '\n== %s ==\n' "$1"
}

print_title "compose status"
for spec in \
  "$LOCAL_BTC_PROJECT:$BTC_COMPOSE_FILE" \
  "$LOCAL_ETH_PROJECT:$ETH_COMPOSE_FILE" \
  "$LOCAL_USDT_PROJECT:$USDT_COMPOSE_FILE" \
  "$LOCAL_SERVICE_PROJECT:$SERVICE_COMPOSE_FILE"; do
  project="${spec%%:*}"
  compose_file="${spec#*:}"

  echo "[$project]"
  if [ -f "$compose_file" ]; then
    dc "$compose_file" "$project" ps -a || true
  else
    echo "compose file missing: $compose_file"
  fi
  echo

done

print_title "artifacts"
for artifact in btc.json eth.json usdt.json smoke-local.json smoke-local-all.json; do
  path="$LOCAL_CHAIN_ARTIFACT_DIR/$artifact"
  if [ -f "$path" ]; then
    echo "$artifact"
    jq '{schema_version, generated_at, network, compose_project, chain_id, contract_address, warnings}' "$path" 2>/dev/null || true
  else
    echo "$artifact (missing)"
  fi
  echo
done
