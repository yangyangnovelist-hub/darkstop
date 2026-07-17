#!/usr/bin/env bash
# Demo helper: simulate FLR/USD dropping below the trigger, then the TEE settle.
# The order flips Pending → Executed in the UI within ~2s.
# Addresses are anvil-deterministic for a fresh `scripts/dev-stack.sh` run.
set -euo pipefail
export PATH="$HOME/.foundry/bin:$PATH"

RPC="http://127.0.0.1:8545"
KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # anvil #0, local only
VAULT="$(grep NEXT_PUBLIC_VAULT_ADDRESS "$(dirname "$0")/../frontend/.env.local" | cut -d= -f2 | tr -d ' ')"
FTSO="0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0" # MockFtsoV2, deterministic

echo "① FLR/USD drops below the \$0.02 trigger..."
cast send "$FTSO" 'setFeed(uint256,int8,uint64)' 150000 7 "$(date +%s)" \
  --rpc-url "$RPC" --private-key "$KEY" >/dev/null
echo "② TEE reveals the trigger and settles on-chain..."
cast send "$VAULT" 'settle(uint256,uint256,uint256)' 1 20000 300 \
  --rpc-url "$RPC" --private-key "$KEY" >/dev/null
echo "✓ Settled. Watch the order flip Pending → Executed in the browser."
