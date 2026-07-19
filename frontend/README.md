# DarkStop frontend

Single-page Next.js UI for the DarkStop confidential stop-loss vault.
Connect a wallet, deposit C2FLR with a trigger price that is **ECIES-encrypted
in the browser** (go-ethereum `crypto/ecies` wire format — see `lib/ecies.ts`),
and watch order status flip live from `OrderPlaced` / `OrderExecuted` /
`OrderCancelled` events.

## Configuration (env)

Defaults target Flare Testnet Coston2 (vault `0xd93E8F7dE2A5A7C4eC45F115f7047103da2dD8bF`,
see `../docs/deployments.md`). Override via `.env.local`:

| Variable | Default | Purpose |
|---|---|---|
| `NEXT_PUBLIC_CHAIN_ID` | `114` | Chain id shown to the wallet |
| `NEXT_PUBLIC_RPC_URL` | Coston2 public RPC | Reads + event watching |
| `NEXT_PUBLIC_EXPLORER_URL` | Coston2 explorer (empty on other chains) | Tx / calldata links |
| `NEXT_PUBLIC_VAULT_ADDRESS` | Coston2 vault | DarkStopVault address |
| `NEXT_PUBLIC_ORDERING_ENABLED` | `false` on Coston2; local dev stack writes `true` | Enables order submission only when the full FCC relay is healthy |
| `NEXT_PUBLIC_START_BLOCK` | probe from genesis, fall back to recent | `getLogs` start for order history |
| `TEE_STATE_URL` | `http://localhost:7702/state` | TEE extension `/state` (proxied server-side by `/api/tee-state`) |
| `DEV_FALLBACK_TEE_PUBKEY` | unset | Local-dev-only pubkey used when the TEE is unreachable |

## Local development stack (simulated enclave, real extension code)

```bash
# from the repo root — starts anvil, deploys mocks + vault, starts Go extension/watcher
./scripts/dev-stack.sh

cd frontend
npm install
npm run dev          # http://localhost:3000
```

Then:

1. Place and relay a deterministic encrypted order through the same TypeScript
   encryption library used by the browser:

   ```bash
   cd frontend
   npx tsx scripts/place-order.ts
   ```

   The script gets the live enclave public key, encrypts the trigger, calls
   `placeOrder`, and relays the official tee-node action/data encoding to the
   running Go extension. The extension decrypts and stores the order.

2. Change only the mock FTSO price:

   ```bash
   cd ..
   ./scripts/demo-settle.sh
   ```

   `demo-settle.sh` never calls `settle()`. The Go watcher observes the price,
   submits settlement, confirms the receipt, and marks enclave state executed.
   The browser order row flips **Pending → Executed** from chain events.

For a terminal-only proof of the full path, run `./scripts/demo-e2e.sh` from a
clean shell. It prints final enclave state and watcher transaction evidence.

Tear down: `./scripts/dev-stack.sh stop`. Delete `frontend/.env.local` to point
the UI back at Coston2.

## Scripts

```bash
npm run dev         # dev server
npm run build       # production build
npm test            # vitest — ECIES conformance suite (lib/ecies.test.ts)
npm run typecheck   # tsc --noEmit
```

## ECIES conformance

`lib/ecies.ts` implements go-ethereum's ECIES (`0x04 ‖ ephPubXY ‖ IV ‖
AES-128-CTR ct ‖ HMAC-SHA-256 tag`, NIST concat-KDF, `Km = SHA-256(K[16:32])`).
**eciesjs is not wire-compatible** (HKDF + AES-GCM) — do not swap it in.
`lib/ecies.test.ts` proves compatibility by decrypting the Go-produced
ciphertext in `internal/extension/testdata/ecies_vector.json` and round-tripping
the encryptor through that validated decryptor.
