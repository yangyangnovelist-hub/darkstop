# DarkStop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Confidential stop-loss orders on Flare — trigger prices live encrypted in a (simulated) TEE extension; settlement is verified on-chain against FTSO.

**Architecture:** Fork of `flare-foundation/fce-extension-scaffold` (vendored read-only copy at `~/Desktop/hackathons/vendor-fce-scaffold`). Three layers: `DarkStopVault.sol` (deposits + FCC instruction sender + FTSO-checked settlement), Go TEE extension (ECIES decrypt, in-enclave order store, FTSO watcher, settle tx submitter), minimal Next.js frontend (client-side ECIES encryption, order status).

**Tech Stack:** Solidity 0.8.27 + Foundry (`flare-foundry-periphery-package` for FTSO), Go 1.25 (scaffold conventions: `processorutils.Parse`, `buildResult`, JSON/ABI decoders), Docker Compose + ngrok (Coston2 mode, `SIMULATED_TEE=true`), Next.js + viem + `eciesjs`.

**Ground truth discovered 2026-07-15 (do not re-litigate):**
- Local dev mode is DEAD for us: it requires the private `gitlab.com/flarenetwork/tee/e2e` repo. All work targets Coston2 directly (`docker-compose.coston2.yaml`, `.env` per `.env.example` which already defaults to Coston2 + `SIMULATED_TEE=true`).
- Coston2 ext-proxy needs **indexer DB credentials** (placeholders in `config/proxy/extension_proxy.coston2.docker.toml.example`) — must be requested in the Flare Hackathon Telegram group (USER ACTION, blocking Phase 0 Task 3).
- Instruction payloads (`message` bytes in `sendInstructions`) are PUBLIC on-chain → confidentiality comes from client-side ECIES encryption to a keypair generated inside the extension (pubkey served via `GET /state`).
- Trigger price is revealed only at settlement (that is fine — post-execution reveal is the product's contract).
- OPType/OPCommand strings must match byte-for-byte across Solidity constants, Go config, and decoder registration (`teeutils.ToHash`).

**Fuse (from design spec):** If Phase 0's hello-world does not run end-to-end on Coston2 within 3 working days of starting, STOP and switch to the PayProof fallback (write a new plan from the design spec's fallback section).

---

## Phase 0 — Environment + Coston2 hello-world spike (the fuse)

### Task 0.1: Install toolchain

- [ ] `brew install go` → verify `go version` ≥ 1.25
- [ ] `curl -L https://foundry.paradigm.xyz | bash && foundryup` → verify `forge --version`
- [ ] `ngrok config check` — if no authtoken, USER signs up at ngrok.com and runs `ngrok config add-authtoken <token>` (Claude must not create the account)
- [ ] Commit nothing (env only)

### Task 0.2: USER ACTIONS (parallel, blocking)

- [ ] User joins Flare Hackathon Telegram `https://t.me/+5Vn6ZKhr6KI3NjIx`, asks (English template provided by Claude) for Coston2 indexer DB credentials for FCC extension development, and confirms judges accept simulated-TEE demos
- [ ] User installs a wallet (Rabby or MetaMask), creates a fresh dev-only account, claims C2FLR at `https://faucet.flare.network/coston2` (captcha — must be human)
- [ ] User exports that dev account's private key into `~/Desktop/hackathons/darkstop/.env` (dev-only key, never reused elsewhere)

### Task 0.3: Deploy unmodified hello-world scaffold to Coston2

Working dir: a scratch copy — `cp -R vendor-fce-scaffold hello-spike && cd hello-spike`

- [ ] `cp .env.example .env`; set `DEPLOYMENT_PRIVATE_KEY`, `INITIAL_OWNER`, `PROXY_PRIVATE_KEY` (same key OK), leave `CHAIN_URL`/`ADDRESSES_FILE`/`LOCAL_MODE=false`/`SIMULATED_TEE=true` as-is
- [ ] `cp config/proxy/extension_proxy.coston2.docker.toml.example config/proxy/extension_proxy.coston2.docker.toml` and fill `[db]` with the credentials from Telegram
- [ ] `ngrok http 6674` (keep running); set `EXT_PROXY_URL=https://<generated>.ngrok-free.dev` in `.env`
- [ ] `./scripts/pre-build.sh` → expect: contract deployed, extension registered, `config/extension.env` written with `EXTENSION_ID` + `INSTRUCTION_SENDER`
- [ ] `docker compose -f docker-compose.yaml -f docker-compose.coston2.yaml up -d --build` → `docker compose ... ps` shows redis, ext-proxy, extension-tee healthy
- [ ] `./scripts/post-build.sh` → TEE version + machine registered on-chain
- [ ] `./scripts/test.sh` → SAY_HELLO instruction round-trips; capture tx hash + proxy result JSON into `docs/spike-notes.md`
- [ ] **FUSE CHECKPOINT**: pass → Phase 1. Fail after 3 days of debugging (use scaffold's `.claude/skills`: `verify-deploy`, `test-extension`) → fallback plan.

## Phase 1 — Bootstrap darkstop repo from scaffold

### Task 1.1: Import scaffold into darkstop repo

- [ ] `rsync -a --exclude .git vendor-fce-scaffold/ darkstop/` (design docs stay; scaffold files land at repo root)
- [ ] `git add -A && git commit -m "chore: import fce-extension-scaffold as base"`

### Task 1.2: Rename HelloWorld → DarkStop

- [ ] Run the scaffold's own Claude skill `/rename-scaffold` (project-scoped) with target name `DarkStop`; follow `docs/manual-setup.md` as checklist: contract `DarkStopInstructionSender` (will become `DarkStopVault`), Go module strings, docker image names, test helpers
- [ ] `cd tools && go test ./...` → PASS; `forge build` → compiles
- [ ] Commit `refactor: rename scaffold to DarkStop`

## Phase 2 — Contracts

Constants used everywhere (Solidity / Go config / decoder registration must match):

```
OP_TYPE_DARKSTOP  = "DARKSTOP"
OP_COMMAND_PLACE  = "PLACE_ORDER"
OP_COMMAND_CANCEL = "CANCEL_ORDER"
```

### Task 2.1: `contracts/DarkStopVault.sol` (replaces InstructionSender, keeps its DO-NOT-MODIFY parts)

Structure (write full contract; key elements):

```solidity
struct Order { address owner; uint256 deposit; uint8 status; } // 0 none,1 open,2 executed,3 cancelled
mapping(uint256 => Order) public orders;
uint256 public nextOrderId;
address public teeExecutor;        // set once by owner after TEE registration
IERC20 public payoutToken;         // testnet USDT0
bytes21 public constant FLR_USD = bytes21(0x01464c522f55534400000000000000000000000000);

function placeOrder(bytes calldata ciphertext) external payable returns (uint256 id);
    // requires msg.value > instructionFee; splits: fee forwarded via sendInstructions
    // (opType DARKSTOP, opCommand PLACE_ORDER, message = abi.encode(PlaceMsg{id, ciphertext}))
    // remainder stored as Order.deposit; emits OrderPlaced(id, owner) — NO price data on chain
function cancel(uint256 id) external; // owner only, refunds deposit, sends CANCEL_ORDER instruction
function settle(uint256 id, uint256 triggerPrice, uint256 maxAgeSec) external; // msg.sender == teeExecutor
    // reads FTSO: (value, decimals, ts) = ContractRegistry.getTestFtsoV2().getFeedById(FLR_USD)
    // require(block.timestamp - ts <= maxAgeSec); require(value normalized <= triggerPrice);
    // payout = deposit * price / 1e(decimals adj); transfer payoutToken to owner; keep deposit; emit OrderExecuted(id, price)
```

- [ ] Add `flare-foundry-periphery-package` as Foundry dependency (`forge install flare-foundation/flare-foundry-periphery-package`, remap in `foundry.toml`)
- [ ] Write Foundry tests FIRST in `test/DarkStopVault.t.sol` with mocked FTSO + mocked registries: place stores deposit & emits no price; settle rejects non-executor / stale price / price above trigger; settle pays correct USDT0 amount and flips status; cancel refunds and blocks double-spend; settle-after-cancel reverts
- [ ] `forge test` red → implement → green
- [ ] Commit each red-green pair (`test: ...` then `feat: ...`)

### Task 2.2: Deploy to Coston2

- [ ] Extend `tools/cmd/deploy-contract` for the new constructor args (payout token: deploy a plain `MockUSDT0.sol` ERC-20 first, mint pool to vault)
- [ ] `./scripts/pre-build.sh` on darkstop repo → record addresses in `docs/deployments.md`; `cast call` sanity: `getFeedById` staleness path works on Coston2 fork test (`forge test --fork-url $COSTON2_RPC` for the FTSO integration test only)
- [ ] Commit `feat: coston2 deployment tooling`

## Phase 3 — Go extension

### Task 3.1: Types + decoders (`pkg/types/types.go`, `pkg/types/register.go`)

```go
type PlaceOrderRequest struct { OrderID *big.Int; Ciphertext []byte }   // ABI tuple mirror of PlaceMsg
type CancelOrderRequest struct { OrderID *big.Int }
type OrderState struct { OrderID string `json:"orderId"`; Status string `json:"status"` } // NO price in state
type State struct { EncryptionPubKey string `json:"encryptionPubKey"`; OpenOrders int `json:"openOrders"`; Orders []OrderState `json:"orders"` }
```

- [ ] ABI `abi.Argument` layouts mirroring the Solidity structs (copy `SayGoodbyeMessageArg` pattern)
- [ ] Register `(DARKSTOP, PLACE_ORDER)` + `(DARKSTOP, CANCEL_ORDER)` decoders with `NewABIDecoder[T](abiStr)`
- [ ] `go test ./pkg/types/...` for decoder round-trip; commit

### Task 3.2: ECIES keypair + order store (`internal/extension/crypto.go`, `store.go`)

- [ ] On startup generate secp256k1 key via `github.com/ethereum/go-ethereum/crypto` + `crypto/ecies` wrapper; expose hex pubkey in `GET /state`
- [ ] Decrypted order = `{TriggerPrice *big.Int, Owner string}` kept ONLY in an in-memory `map[uint64]Order` behind `sync.RWMutex` (simulated-TEE mode; restart-recovery documented as roadmap, not built — YAGNI)
- [ ] Unit tests: encrypt-with-eciesjs-compatible-format → decrypt round trip (test vector generated once with eciesjs, committed as fixture); commit

### Task 3.3: Handlers (`internal/extension/extension.go`)

- [ ] `processDarkstop` routing per scaffold's `processGreeting` pattern; `processPlaceOrder`: decode ABI → decrypt ciphertext → validate trigger > 0 → store → `buildResult(action, df, {"orderId":...,"status":"open"}, 1, nil)`; `processCancelOrder`: drop from store
- [ ] Table-driven unit tests with fixture ciphertexts (bad ciphertext, zero trigger, duplicate id, happy path); commit each red-green

### Task 3.4: FTSO watcher + settler (`internal/extension/watcher.go`)

- [ ] Goroutine every 5s: `ethclient.Dial(CHAIN_URL)` → call FtsoV2 `getFeedById(FLR_USD)` (bindings via `./scripts/generate-bindings.sh` pointed at the periphery ABI) → for each open order where price ≤ trigger: send `settle(id, trigger, 300)` from `EXECUTOR_PRIVATE_KEY` (new env var; same funded dev key OK), gas via `SuggestGasPrice`, retry ×3 with backoff, mark executed on receipt success, log every attempt (this log IS the audit-trail demo material)
- [ ] Unit test with mocked ethclient interface for the trigger comparison + retry logic; integration path tested in Phase 4 instead of mocks-for-everything; commit

## Phase 4 — End-to-end on Coston2

- [ ] Update `tools/cmd/run-test`: payload = eciesjs-fixture ciphertext with trigger set ABOVE current spot (instant trigger) → assert proxy result `status:"open"`, then poll vault until `OrderExecuted`
- [ ] Full run: `./scripts/pre-build.sh` → compose up (coston2 files) → `./scripts/post-build.sh` → owner calls `setTeeExecutor` → `./scripts/test.sh`
- [ ] Second manual order with realistic below-spot trigger left running; capture explorer links for both
- [ ] Record all tx hashes + screenshots into `docs/demo-evidence.md`; commit

## Phase 5 — Frontend (`frontend/`, Next.js single page)

- [ ] `npx create-next-app@latest frontend --ts --tailwind --app --no-src-dir`; deps: `viem`, `wagmi`, `@tanstack/react-query`, `@noble/curves`
- [ ] **eciesjs is NOT wire-compatible with go-ethereum ECIES (verified in Phase 3)** — write a ~40-line custom encryptor (@noble/curves secp256k1 + WebCrypto AES-128-CTR + HMAC-SHA-256, NIST concat-KDF): output `0x04‖ephemeralPubXY(64)‖IV(16)‖ct‖HMAC tag(32)`, z = shared X (32B), K = concatKDF(SHA-256, z, nil, 32), Ke=K[:16], Km=SHA-256(K[16:32]), tag=HMAC(Km, IV‖ct). Conformance target: `internal/extension/testdata/ecies_vector.json` — add a vitest that reproduces the fixture ciphertext structure and round-trips against a Go decrypt helper if available.
- [ ] One page: connect wallet (Coston2 chain config) → fetch `GET <ngrok>/state`-proxied pubkey (via a `/api/tee-state` route to dodge CORS) → form (amount C2FLR, trigger USD) → encrypt `{triggerPrice}` with eciesjs → `placeOrder(ciphertext)` with value → live order table from `OrderPlaced`/`OrderExecuted` events (viem `watchContractEvent`), status chips Pending/Executed/Cancelled
- [ ] Explicit copy on page: "Your trigger price never touches the chain — inspect the tx yourself" + link to the calldata in explorer (the pitch, in-product)
- [ ] Manual E2E per superpowers verification: place a real order from the browser on Coston2, watch it execute; commit

## Phase 6 — Submission package (deadline 2026-08-14, buffer from 08-10)

- [ ] English README rewrite: problem, architecture diagram (mermaid), trust model honesty section (simulated TEE now → Songbird real TEE roadmap), what-was-built-during-hackathon statement (everything; base scaffold credited), run-it-yourself guide
- [ ] 3-minute demo video script (English, scene-by-scene: the front-run problem, place encrypted order, show explorer calldata is opaque, price crosses, auto-execution, FTSO re-check in settle code) — user records following script
- [ ] User: create public GitHub repo, push, submit BUIDL on DoraHacks with contract addresses, video, repo, roadmap
- [ ] Post-submission: ask a question in the DoraHacks "Ask Question" tab confirming receipt

---

## Self-review notes

- Spec coverage: problem/solution (P2, P3, P5), FTSO dual-use (2.1 settle + 3.4 watcher), FCC full lifecycle (0.3, 2.2, 4), fallback fuse (Phase 0), submission checklist (Phase 6), division of labor (0.2, 6). Cancel path covered 2.1/3.3. Error handling: staleness + retries (2.1, 3.4).
- Known discovery points left deliberately open (resolved by running, not guessing): exact `pre-build.sh` behavior against a renamed contract, bindings output shape, eciesjs↔go-ecies parameter compatibility (fixture test settles it early in 3.2).
- Scope: one pair, one order type, no persistence, no real TEE — matches spec's YAGNI cuts.
