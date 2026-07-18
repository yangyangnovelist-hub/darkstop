# DoraHacks BUIDL Submission — Pre-filled Fields

Copy-paste each field below into the DoraHacks BUIDL form for **Flare Summer
Signal**. Replace the remaining `TODO` placeholder (demo video URL) before
submitting. The public repository URL is already final.

---

## Project name

```
DarkStop
```

## One-liner / tagline

```
Confidential stop-loss orders on Flare: trigger prices live encrypted in a TEE, never on-chain, and every settlement is re-verified against FTSO.
```

## Bounty / track selection

```
Bounty 2 — Confidential Compute Apps
```

## Description

```
On-chain stop-loss orders leak the one thing they must protect: the trigger price. It sits in public contract storage or a keeper's mempool, so anyone can read every trader's liquidation level, push the price to it, and absorb the forced sell. Centralized exchanges hide your stops; DeFi today cannot.

DarkStop fixes this with Flare Confidential Compute. The trigger price is ECIES-encrypted in the browser to the TEE extension's enclave key. On-chain, placeOrder(ciphertext) stores only the deposit and an opaque blob — inspect the calldata yourself; there is no price anywhere. The ciphertext is forwarded through the FCC instruction flow to a Go TEE extension that decrypts it, keeps the order in enclave memory, and watches the FTSO FLR/USD feed. When the price crosses the trigger, the extension submits settle() — and the contract still does not trust the TEE alone: settle() re-reads the live FTSO feed on-chain and requires a fresh price at-or-below the revealed trigger before paying out in USDT0.

What works today: the vault is deployed and registered on Coston2 (extension id 503, TEE machine registered on-chain); one command runs the full local encrypted place → Go extension decrypt/store → FTSO trigger → real watcher settle → enclave/chain consistency proof; Coston2 fork tests place and settle orders against the real live FTSO feed; and the browser encryptor is proven wire-compatible with go-ethereum's ECIES by a cross-language conformance suite. 114 tests total (21 Foundry unit + 4 Coston2 fork + 78 Go + 11 frontend).

Honesty section: the TEE runs in simulated mode — the mode the official scaffold supports for Coston2, and the Flare team confirmed in the hackathon Telegram that the Coston2 simulated approach is accepted for judging. Live-testnet placeOrder is currently gated on the Flare-side FTDC proxy producing an availability proof for our registered machine (our proxy is provably healthy — Flare's proxy polls its TEE_INFO every ~10s); this is documented and escalated. Full account in docs/coston2-runbook.md.
```

## Demo video link

```
TODO — YouTube (unlisted) URL after recording per docs/demo-video-script.md
```

## GitHub repo link

```
https://github.com/yangyangnovelist-hub/darkstop
```

## How the project uses Flare

```
FCC (Flare Confidential Compute) — full lifecycle, not a toy call: DarkStopVault is a real FCC instruction sender. Extension registered on TeeExtensionRegistry (id 503), instruction fee paid through sendInstructions, OPType/OPCommand (DARKSTOP / PLACE_ORDER / CANCEL_ORDER) mirrored byte-for-byte across Solidity constants, Go config, and decoder registration, TEE machine registered on-chain. The product is impossible as a plain smart contract — the confidentiality is the product, and it comes from the TEE.

FTSO — used in two places:
1. Inside the TEE: the watcher polls the block-latency FLR/USD feed to detect trigger crossings privately, so monitoring leaks nothing.
2. On-chain at settlement: settle() calls FtsoV2.getFeedById(FLR_USD) itself and requires the price to be fresh and at-or-below the revealed trigger. The executor may request a stricter window, but the contract rejects anything above 300 seconds. The contract never trusts the TEE's price report alone — FTSO is the on-chain arbiter.
```

## What was built during the hackathon (new work statement)

```
Everything in the repository is new work built during the hackathon. The base is the official flare-foundation/fce-extension-scaffold (the Hello World FCC template, credited in the README); on top of it we built: DarkStopVault.sol (deposit vault + FCC instruction sender + FTSO-verified settlement + cancel/refund), the Go TEE extension logic (ECIES decryption, in-enclave order store, PLACE_ORDER/CANCEL_ORDER handlers, FTSO watcher with retry/backoff and audit logging), a Next.js frontend with a from-scratch browser ECIES encryptor wire-compatible with go-ethereum (eciesjs is not — we verified, documented, and wrote our own with a cross-language conformance suite), Coston2 deployment tooling, a one-shot local dev stack, fork tests against the live Coston2 FTSO, and a Coston2 TEE bring-up runbook. No prior codebase existed.
```

## Contract addresses (Coston2, chain id 114)

```
DarkStopVault (instruction sender): 0xd93E8F7dE2A5A7C4eC45F115f7047103da2dD8bF
https://coston2-explorer.flare.network/address/0xd93E8F7dE2A5A7C4eC45F115f7047103da2dD8bF

MockUSDT0 (payout token): 0x6196b20FaeCE88ace220297122bB170A5B97b60F
https://coston2-explorer.flare.network/address/0x6196b20FaeCE88ace220297122bB170A5B97b60F

Extension ID on TeeExtensionRegistry: 503
FtsoV2 (resolved via FlareContractRegistry): 0xC4e9c78EA53db782E28f28Fdf80BaF59336B304d
```

## Roadmap

```
1. Real attested TEE on Songbird once the FCC rollout (STP.13) completes — replaces simulated mode with hardware attestation.
2. Live Coston2 placeOrder as soon as the Flare FTDC proxy produces the availability proof for our registered machine (all our-side registration steps are complete).
3. Multiple trading pairs and take-profit / trailing-stop order types.
4. Real DEX settlement hop instead of the pre-funded testnet payout pool.
5. Enclave state recovery by replaying OrderPlaced events after extension restart.
```

---

## 提交时的操作提醒（中文，不要粘贴进表单）

1. 确认公开仓库 `https://github.com/yangyangnovelist-hub/darkstop` 已同步最新 `main`。
2. 按 `docs/demo-video-script.md` 录好视频，传 YouTube（Unlisted），URL 填第一个 TODO。
3. DoraHacks 上确认选择的是 Flare Summer Signal 活动下的 Bounty 2（Confidential Compute）。
4. 提交后到 BUIDL 页面的 "Ask Question" 标签发一条留言确认收到（例如 "Submitted for Bounty 2 — please confirm receipt"）。
5. 截止日期 2026-08-14，建议 08-10 前提交留缓冲。
