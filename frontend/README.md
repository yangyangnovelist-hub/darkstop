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
| `NEXT_PUBLIC_START_BLOCK` | probe from genesis, fall back to recent | `getLogs` start for order history |
| `TEE_STATE_URL` | `http://localhost:7702/state` | TEE extension `/state` (proxied server-side by `/api/tee-state`) |
| `DEV_FALLBACK_TEE_PUBKEY` | unset | Local-dev-only pubkey used when the TEE is unreachable |

## Local development stack (no testnet, no real TEE)

```bash
# from the repo root — starts anvil, deploys mocks + vault, writes .env.local
./scripts/dev-stack.sh

cd frontend
npm install
npm run dev          # http://localhost:3000
```

Then:

1. In MetaMask add a network: RPC `http://127.0.0.1:8545`, chain id `31337`,
   and import anvil key #0
   (`0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80`).
2. Place an order in the UI (e.g. deposit `1`, trigger `0.02`). The trigger is
   encrypted to the repo's ECIES fixture key (`internal/extension/testdata/
   ecies_vector.json`) because no TEE extension runs locally — the `/api/tee-state`
   proxy falls back to `DEV_FALLBACK_TEE_PUBKEY`. If you run the real extension
   on port 7702, the proxy uses its live enclave key instead.
3. Simulate the price crossing and the TEE settle (addresses are printed by
   `dev-stack.sh` and recorded as comments in `.env.local`):

   ```bash
   export PATH="$HOME/.foundry/bin:$PATH"
   # drop FLR/USD to $0.015 with a fresh timestamp
   cast send $FTSO 'setFeed(uint256,int8,uint64)' 150000 7 $(date +%s) \
     --rpc-url http://127.0.0.1:8545 --private-key <anvil key #0>
   # TEE executor (= deployer locally) reveals trigger 20000 and settles order 1
   cast send $VAULT 'settle(uint256,uint256,uint256)' 1 20000 300 \
     --rpc-url http://127.0.0.1:8545 --private-key <anvil key #0>
   ```

4. The order row flips **Pending → Executed** in the UI within ~2s (event
   polling). `cancel` from the UI flips it to **Cancelled** and refunds.

Tear down: `./scripts/dev-stack.sh stop` (and delete `frontend/.env.local`
to point the UI back at Coston2).

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
