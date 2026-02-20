#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/local-chains/common.sh
source "$SCRIPT_DIR/common.sh"

require_cmd jq
require_cmd go

DEVTEST_KEYSETS_JSON="${PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON:-}"

if [ -z "$DEVTEST_KEYSETS_JSON" ]; then
  echo "missing PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON for startup keyset preflight" >&2
  exit 1
fi

if ! printf '%s' "$DEVTEST_KEYSETS_JSON" | jq -e 'type == "object"' >/dev/null 2>&1; then
  echo "invalid PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON: must be a JSON object" >&2
  exit 1
fi

derive_address_scheme() {
  local chain="$1"
  case "$chain" in
    bitcoin)
      printf 'bip84_p2wpkh'
      ;;
    ethereum)
      printf 'evm_bip44'
      ;;
    *)
      return 1
      ;;
  esac
}

keyset_rows="$(
  printf '%s' "$DEVTEST_KEYSETS_JSON" | jq -c '
    to_entries[]
    | select((.value | type) == "object")
    | .key as $chain
    | .value
    | to_entries[]
    | select((.value | type) == "object")
    | {
        chain: ($chain | ascii_downcase),
        network: (.key | ascii_downcase),
        keyset_id: (.value.keyset_id // ""),
        extended_public_key: (.value.extended_public_key // .value.key_material // .value.xpub // ""),
        expected_index0_address: (.value.expected_index0_address // "")
      }
  '
)"

if [ -z "$keyset_rows" ]; then
  echo "startup keyset preflight requires nested keysets format (chain -> network -> entry object)" >&2
  exit 1
fi

verified_count=0

while IFS= read -r row; do
  [ -n "$row" ] || continue

  chain="$(printf '%s' "$row" | jq -r '.chain')"
  network="$(printf '%s' "$row" | jq -r '.network')"
  keyset_id="$(printf '%s' "$row" | jq -r '.keyset_id')"
  extended_public_key="$(printf '%s' "$row" | jq -r '.extended_public_key')"
  expected_index0_address="$(printf '%s' "$row" | jq -r '.expected_index0_address')"

  if [ -z "$chain" ] || [ -z "$network" ] || [ -z "$keyset_id" ] || [ -z "$extended_public_key" ] || [ -z "$expected_index0_address" ]; then
    echo "startup keyset preflight invalid entry: chain/network/keyset_id/extended_public_key/expected_index0_address are required" >&2
    echo "$row" >&2
    exit 1
  fi

  if ! address_scheme="$(derive_address_scheme "$chain")"; then
    echo "startup keyset preflight unsupported chain=$chain network=$network keyset_id=$keyset_id" >&2
    exit 1
  fi

  if verify_output="$(
    cd "$REPO_ROOT"
    go run ./cmd/keysetverify \
      --chain "$chain" \
      --network "$network" \
      --address-scheme "$address_scheme" \
      --keyset-id "$keyset_id" \
      --extended-public-key "$extended_public_key" \
      --expected-address "$expected_index0_address" 2>&1
  )"; then
    echo "startup keyset preflight passed chain=$chain network=$network keyset_id=$keyset_id"
  else
    echo "startup keyset preflight failed chain=$chain network=$network keyset_id=$keyset_id" >&2
    printf '%s\n' "$verify_output" >&2
    exit 1
  fi

  verified_count=$((verified_count + 1))
done <<< "$keyset_rows"

if [ "$verified_count" -eq 0 ]; then
  echo "startup keyset preflight found no verifiable keysets" >&2
  exit 1
fi

echo "startup keyset preflight completed verified=$verified_count"
