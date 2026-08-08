#!/usr/bin/env bash
# One-command local proof of the real DarkStop execution path:
# encrypted order -> Go extension decrypt/store -> FTSO drop -> Go watcher settle.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"$ROOT/scripts/dev-stack.sh"

# The trailing policy is advertised only while the watcher has a fresh FTSO
# sample. Building the Go binary can consume the mock feed's initial 10-second
# freshness window, so refresh the local feed immediately before placement.
RPC="http://127.0.0.1:8545"
DEV_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
FTSO="$(grep DEV_FTSO_ADDRESS "$ROOT/frontend/.env.local" | cut -d= -f2 | tr -d ' ')"
cast send "$FTSO" 'setFeed(uint256,int8,uint64)' 300000 7 "$(date +%s)" \
  --rpc-url "$RPC" --private-key "$DEV_KEY" >/dev/null
for _ in $(seq 1 20); do
  curl -fsS http://127.0.0.1:7702/state | jq -e \
    '.state.supportedPolicies | index("trailing") != null' >/dev/null && break
  sleep 0.25
done
curl -fsS http://127.0.0.1:7702/state | jq -e \
  '.state.supportedPolicies | index("trailing") != null' >/dev/null

(cd "$ROOT/frontend" && npx tsx scripts/place-order.ts)
"$ROOT/scripts/demo-guardrails.sh"
"$ROOT/scripts/demo-settle.sh"

echo
echo "TEE state after settlement:"
curl -fsS http://127.0.0.1:7702/state | jq '{openOrders: .state.openOrders, orders: .state.orders}'
echo
echo "Watcher proof:"
grep -E 'TRIGGERED|settle tx sent|settle confirmed|marked executed|local state reconciled|fee-bumped' /tmp/darkstop-extension.log | tail -n 8
