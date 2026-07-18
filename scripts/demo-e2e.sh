#!/usr/bin/env bash
# One-command local proof of the real DarkStop execution path:
# encrypted order -> Go extension decrypt/store -> FTSO drop -> Go watcher settle.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"$ROOT/scripts/dev-stack.sh"
(cd "$ROOT/frontend" && npx tsx scripts/place-order.ts)
"$ROOT/scripts/demo-settle.sh"

echo
echo "TEE state after settlement:"
curl -fsS http://127.0.0.1:7702/state | jq '{openOrders: .state.openOrders, orders: .state.orders}'
echo
echo "Watcher proof:"
grep -E 'TRIGGERED|settle tx sent|settle confirmed|marked executed' /tmp/darkstop-extension.log | tail -n 8
