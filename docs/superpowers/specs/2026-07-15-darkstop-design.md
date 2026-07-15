# DarkStop — Confidential Stop-Loss Orders on Flare (Design)

Date: 2026-07-15
Competition: Flare Summer Signal (DoraHacks), Bounty 2 — Confidential Compute Apps
Deadline: 2026-08-14 (submission), judging 08-15–21
Prize target: $6,000 track pool (1st $4,000 / 2nd $2,000)

## Problem

On-chain stop-loss and limit orders leak the trader's liquidation levels: trigger
prices sit in public contract storage or public keeper mempools, so predators can
hunt stops (push price to the trigger, absorb the forced sell). Centralized
exchanges hide your stops; DeFi today cannot.

## Solution (one sentence)

Stop-loss orders whose trigger conditions live encrypted inside a TEE (Flare
Confidential Compute); the chain only ever sees "an order exists" and, at
execution time, a TEE-signed settlement verified against the FTSO price.

## Why Flare specifically (judging: "meaningful integration")

- **FCC** (InstructionSender pattern, TeeExtensionRegistry, OPType/OPCommand
  routing, TEE signature verification) — the confidentiality is impossible in a
  plain smart contract.
- **FTSO** used twice: block-latency feed read inside the TEE extension for
  trigger monitoring, and on-chain `FtsoV2Interface.getFeedById()` re-check at
  settlement so the contract never trusts the TEE alone for price.

## Scope (deliberately small)

- One trading pair: FLR/USD (FTSO feed `0x01464c522f555344...`).
- One order type: stop-loss sell (trigger when price <= threshold).
- Deployment: Coston2 testnet (chain id 114), TEE simulated locally per the
  official scaffold's supported mode (`docker-compose.coston2.yaml` + ngrok).
- Out of scope: multiple pairs, take-profit/limit orders, real GCP Confidential
  Space deployment, mainnet, order cancellation UI polish (cancel exists as a
  contract call only).

## Architecture

### 1. Contracts (Solidity, Hardhat, based on flare-hardhat-starter)

- `DarkStopVault.sol`
  - `placeOrder(bytes encryptedOrder) payable` — user deposits C2FLR (the asset
    to be sold) plus an encrypted blob (trigger price + min payout encrypted to
    the TEE extension's public key). Emits `OrderPlaced(orderId, owner)`.
    Nothing price-related is readable on chain.
  - `settle(orderId, price, teeSig)` — called via the FCC instruction flow;
    verifies (a) TEE signature via the registered extension key, (b) FTSO
    current price is at-or-below the TEE-reported trigger within a staleness
    window, then swaps the deposit to the stable side (testnet: transfers
    deposit to a mock treasury and pays the owner in test USDT0 from a
    pre-funded pool — a real DEX hop is out of scope) and emits
    `OrderExecuted`.
  - `cancel(orderId)` — owner reclaims deposit if not executed.
- `InstructionSender` wiring per the fce-extension-scaffold conventions
  (OPType/OPCommand constants mirrored in the Go config).

### 2. TEE extension (Go, from flare-foundation/fce-extension-scaffold)

- Handler `PLACE_ORDER`: decrypt order blob, validate, store in enclave memory
  (backed by the scaffold's Redis inside the trust boundary for the simulated
  mode).
- Watcher loop: poll FTSO block-latency FLR/USD each block-ish interval; when
  trigger hit, sign settlement payload (orderId, observed price, timestamp) and
  submit the settle instruction on Coston2.
- Handler `CANCEL_ORDER`: drop enclave state on on-chain cancel event.

### 3. Frontend (minimal Next.js single page)

- Connect wallet (Coston2), place an encrypted stop order (encryption happens
  client-side with the extension's published pubkey), watch order status flip
  Pending → Executed live. This is the demo's money shot.

## Data flow

user (browser encrypts order) → `placeOrder` on Coston2 → FCC instruction →
TEE extension decrypts & stores → FTSO price crosses trigger → extension signs
& submits settle → `settle()` re-verifies price on-chain via FTSO → payout →
frontend shows execution + Coston2 explorer link.

## Error handling

- Extension restart: enclave state rebuilt by replaying `OrderPlaced` events
  and re-requesting blobs (simulated-TEE mode keeps Redis, so this is a
  recovery path, documented not demoed).
- Stale/failed settle tx: retry with backoff; settle is idempotent per orderId.
- FTSO staleness: settle rejects prices older than N seconds.
- Fallback plan (fuse): if the scaffold's full-setup + Coston2 mode is not
  demonstrably working by day 3 of build, switch tracks to the FDC PayProof
  design (cross-chain payment receipts), which has low technical risk.

## Testing

- Hardhat unit tests: vault accounting, signature verification, FTSO staleness
  and threshold checks, cancel/settle races.
- Scaffold's `./scripts/full-setup.sh --test` end-to-end locally.
- Manual end-to-end on Coston2 with faucet C2FLR: place → trigger (wait for a
  real FTSO price cross, or use a test-only order with trigger near spot) →
  executed, captured for the demo video.

## Submission checklist (from hackathon rules)

Public GitHub repo (English README: what existed before = nothing, all new
work), demo video, working app link, explanation of Flare usage, contract
addresses on Coston2, short roadmap (real TEE deployment on Songbird once FCC
STP.13 rollout completes; multi-pair; take-profit orders).

## Division of labor

Claude writes all code, docs, and the demo script (English). User handles:
DoraHacks registration/submission, wallet installation, faucet claims, and
recording the demo video following the provided script.
