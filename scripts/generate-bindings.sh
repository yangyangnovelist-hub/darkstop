#!/usr/bin/env bash
# generate-bindings.sh — Compile Solidity contracts and generate Go bindings.
#
# Generates bindings for:
#   - DarkStopVault (contracts/DarkStopVault.sol) — vault + instruction sender
#   - MockUSDT0     (contracts/MockUSDT0.sol)     — testnet payout token
#
# Prerequisites: forge (Foundry), jq
#
# Usage: ./scripts/generate-bindings.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

GO_PKG="darkstop"
BINDINGS_DIR="$PROJECT_DIR/tools/pkg/contracts/$GO_PKG"

cd "$PROJECT_DIR"

echo "=== Step 1: Compile Solidity contracts ==="
forge build

# extract_contract <ContractName> <SourceFile.sol>
extract_contract() {
    local name="$1" src="$2"

    if ! grep -q "contract ${name}" "$PROJECT_DIR/contracts/${src}" 2>/dev/null; then
        echo ""
        echo "ERROR: Contract name '${name}' not found in contracts/${src}."
        echo "Make sure the contract name matches this script."
        exit 1
    fi

    local forge_out="$PROJECT_DIR/out/${src}/${name}.json"
    if [[ ! -f "$forge_out" ]]; then
        echo "ERROR: forge output not found at $forge_out"
        exit 1
    fi

    jq '.abi' "$forge_out" > "$BINDINGS_DIR/${name}.abi"
    jq -r '.bytecode.object' "$forge_out" | sed 's/^0x//' > "$BINDINGS_DIR/${name}.bin"

    echo "  ABI → $BINDINGS_DIR/${name}.abi"
    echo "  BIN → $BINDINGS_DIR/${name}.bin"
}

echo "=== Step 2: Extract ABI and BIN ==="
mkdir -p "$BINDINGS_DIR"
extract_contract "DarkStopVault" "DarkStopVault.sol"
extract_contract "MockUSDT0" "MockUSDT0.sol"

# Drop stale bindings from the pre-vault contract, if present.
rm -f "$BINDINGS_DIR/DarkStopInstructionSender.abi" "$BINDINGS_DIR/DarkStopInstructionSender.bin"

echo "=== Step 3: Generate Go bindings ==="
cd "$PROJECT_DIR/tools"
go generate ./pkg/contracts/$GO_PKG/

echo "=== Done ==="
echo "Generated: $BINDINGS_DIR/autogen.go, $BINDINGS_DIR/autogen_usdt0.go"
