#!/usr/bin/env bash
# Read-only eth_call proofs that invalid settlement attempts are rejected.
# Run after placing order #1 and before scripts/demo-settle.sh.
set -euo pipefail

RPC="http://127.0.0.1:8545"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VAULT="$(grep NEXT_PUBLIC_VAULT_ADDRESS "$ROOT/frontend/.env.local" | cut -d= -f2 | tr -d ' ')"
EXECUTOR="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
OUTSIDER="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"

expect_revert() {
  local label="$1"
  local expected="$2"
  shift 2
  local output
  if output="$("$@" 2>&1)"; then
    echo "✗ $label unexpectedly succeeded" >&2
    exit 1
  fi
  if [[ "$output" != *"$expected"* ]]; then
    echo "✗ $label failed for the wrong reason: $output" >&2
    exit 1
  fi
  echo "✓ BLOCKED — $label ($expected)"
}

expect_revert "outsider tries to settle" "not tee executor" \
  cast call "$VAULT" 'settle(uint256,uint256,uint256)' 1 28500 300 \
  --from "$OUTSIDER" --rpc-url "$RPC"

expect_revert "executor tries before the private condition" "price above trigger" \
  cast call "$VAULT" 'settle(uint256,uint256,uint256)' 1 28500 300 \
  --from "$EXECUTOR" --rpc-url "$RPC"

echo "Two independent on-chain guardrails rejected invalid execution."
