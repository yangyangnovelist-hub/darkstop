// Places a DarkStop order from the CLI (stands in for the MetaMask flow during
// the demo). Encrypts the trigger price to the local dev-stack's fixture TEE key
// with the SAME lib the browser uses, then calls placeOrder from anvil key #0.
//
//   cd frontend && npx tsx scripts/place-order.ts
//
// Reads config from .env.local (vault address, RPC, fixture pubkey).
import { createWalletClient, createPublicClient, http, parseEther } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { readFileSync } from "node:fs";
import { encryptTriggerPrice } from "../lib/ecies";
import { vaultAbi } from "../lib/vault";

const env = Object.fromEntries(
  readFileSync(new URL("../.env.local", import.meta.url), "utf8")
    .split("\n")
    .filter((l) => l.includes("="))
    .map((l) => [l.slice(0, l.indexOf("=")).trim(), l.slice(l.indexOf("=") + 1).trim()]),
);

const RPC = env.NEXT_PUBLIC_RPC_URL;
const VAULT = env.NEXT_PUBLIC_VAULT_ADDRESS as `0x${string}`;
const TEE_PUBKEY = env.DEV_FALLBACK_TEE_PUBKEY;
// anvil funded account #0 — local dev only, never a real key.
const ANVIL_KEY = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";

const DEPOSIT = parseEther("1");          // 1 FLR to protect
const TRIGGER_6DEC = "20000";             // 0.02 USD/FLR × 1e6

async function main() {
  const account = privateKeyToAccount(ANVIL_KEY);
  const pub = createPublicClient({ transport: http(RPC) });
  const wallet = createWalletClient({ account, transport: http(RPC) });

  // Dev-stack instruction fee is a fixed 0.01 ETH (verified via `cast call INSTRUCTION_FEE`).
  const fee = parseEther("0.01");

  const ciphertext = encryptTriggerPrice(TEE_PUBKEY, TRIGGER_6DEC);
  console.log(`Encrypted trigger $0.02 → ${ciphertext.slice(0, 42)}… (${(ciphertext.length - 2) / 2} bytes)`);

  const hash = await wallet.writeContract({
    address: VAULT, abi: vaultAbi, functionName: "placeOrder",
    args: [ciphertext], value: DEPOSIT + fee, chain: null,
  });
  console.log(`placeOrder tx: ${hash}`);
  const rcpt = await pub.waitForTransactionReceipt({ hash });
  console.log(`mined in block ${rcpt.blockNumber}, status ${rcpt.status}`);
  console.log("→ the order should now appear as Pending in the UI.");
}

main().catch((e) => { console.error(e); process.exit(1); });
