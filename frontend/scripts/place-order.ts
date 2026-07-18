// Places a DarkStop order from the CLI (stands in for the MetaMask flow during
// the demo). Encrypts the trigger price to the local dev-stack's fixture TEE key
// with the SAME lib the browser uses, then calls placeOrder from anvil key #0.
//
//   cd frontend && npx tsx scripts/place-order.ts
//
// Reads config from .env.local (vault address, RPC, fixture pubkey).
import {
  createWalletClient,
  createPublicClient,
  encodeAbiParameters,
  http,
  parseEther,
  parseEventLogs,
  stringToHex,
  toHex,
} from "viem";
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
const TEE_STATE_URL = env.TEE_STATE_URL;
const TEE_ACTION_URL = env.TEE_ACTION_URL;
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

  const stateRes = await fetch(TEE_STATE_URL);
  if (!stateRes.ok) throw new Error(`TEE state returned ${stateRes.status}`);
  const stateBody = await stateRes.json();
  const teePubkey = stateBody?.state?.encryptionPubKey;
  if (!teePubkey) throw new Error("TEE state has no encryptionPubKey");

  const ciphertext = encryptTriggerPrice(teePubkey, TRIGGER_6DEC);
  console.log(`Encrypted trigger $0.02 → ${ciphertext.slice(0, 42)}… (${(ciphertext.length - 2) / 2} bytes)`);

  const hash = await wallet.writeContract({
    address: VAULT, abi: vaultAbi, functionName: "placeOrder",
    args: [ciphertext], value: DEPOSIT + fee, chain: null,
  });
  console.log(`placeOrder tx: ${hash}`);
  const rcpt = await pub.waitForTransactionReceipt({ hash });
  console.log(`mined in block ${rcpt.blockNumber}, status ${rcpt.status}`);

  const placed = parseEventLogs({
    abi: vaultAbi,
    logs: rcpt.logs,
    eventName: "OrderPlaced",
  })[0];
  if (!placed) throw new Error("placeOrder receipt has no OrderPlaced event");

  const originalMessage = encodeAbiParameters(
    [{ type: "uint256" }, { type: "bytes" }],
    [placed.args.orderId, ciphertext],
  );
  const dataFixed = {
    instructionId: `0x${"12".repeat(32)}`,
    teeId: `0x${"7e".repeat(20)}`,
    timestamp: 0,
    rewardEpochId: 0,
    opType: stringToHex("DARKSTOP", { size: 32 }),
    opCommand: stringToHex("PLACE_ORDER", { size: 32 }),
    cosigners: [],
    cosignersThreshold: 0,
    originalMessage,
  };
  const action = {
    data: {
      id: `0x${"34".repeat(32)}`,
      type: "instruction",
      submissionTag: "submit",
      message: toHex(JSON.stringify(dataFixed)),
    },
    additionalVariableMessages: [],
    timestamps: [],
    additionalActionData: "0x",
    signatures: [],
  };
  const actionRes = await fetch(TEE_ACTION_URL, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(action),
  });
  const actionResult = await actionRes.json();
  if (!actionRes.ok || actionResult?.status !== 1) {
    throw new Error(`TEE rejected instruction: ${JSON.stringify(actionResult)}`);
  }
  console.log(`TEE decrypted and accepted order #${placed.args.orderId}; watcher is live.`);
}

main().catch((e) => { console.error(e); process.exit(1); });
