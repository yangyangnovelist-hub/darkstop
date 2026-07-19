#!/usr/bin/env bash
# Demo helper: drop FLR/USD below the trigger, then wait for the real Go watcher
# to observe the feed and submit settle(). This script never calls settle itself.
# Addresses are anvil-deterministic for a fresh `scripts/dev-stack.sh` run.
set -euo pipefail
export PATH="$HOME/.foundry/bin:$PATH"

RPC="http://127.0.0.1:8545"
KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # anvil #0, local only
VAULT="$(grep NEXT_PUBLIC_VAULT_ADDRESS "$(dirname "$0")/../frontend/.env.local" | cut -d= -f2 | tr -d ' ')"
FTSO="$(grep DEV_FTSO_ADDRESS "$(dirname "$0")/../frontend/.env.local" | cut -d= -f2 | tr -d ' ')"

echo "① Refreshing FLR/USD at \$0.03 so the TEE can seed its private high-watermark..."
cast send "$FTSO" 'setFeed(uint256,int8,uint64)' 300000 7 "$(date +%s)" \
  --rpc-url "$RPC" --private-key "$KEY" >/dev/null
for _ in $(seq 1 12); do
  if grep -q 'FLR/USD = 30000.*1 open order' /tmp/darkstop-extension.log; then
    break
  fi
  sleep 1
done
if ! grep -q 'FLR/USD = 30000.*1 open order' /tmp/darkstop-extension.log; then
  echo "watcher did not seed the trailing high-watermark within 12s" >&2
  exit 1
fi

echo "② FLR/USD falls 50% through the hidden 5% trailing boundary..."
cast send "$FTSO" 'setFeed(uint256,int8,uint64)' 150000 7 "$(date +%s)" \
  --rpc-url "$RPC" --private-key "$KEY" >/dev/null
echo "③ Real Go TEE watcher will settle automatically..."
for _ in $(seq 1 20); do
  STATUS="$(cast call "$VAULT" 'orders(uint256)(address,uint256,uint8)' 1 --rpc-url "$RPC" | tail -n 1 | awk '{print $1}')"
  TEE_STATUS="$(curl -fsS http://127.0.0.1:7702/state 2>/dev/null | jq -r '.state.orders[] | select(.orderId == "1") | .status' || true)"
  if [[ "$STATUS" == "2" && "$TEE_STATUS" == "executed" ]]; then
    echo "✓ Private trailing condition met. FTSO verified settlement on-chain."
    exit 0
  fi
  sleep 1
done

echo "watcher did not settle order #1 within 20s; see /tmp/darkstop-extension.log" >&2
exit 1
