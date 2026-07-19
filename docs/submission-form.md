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
Private trailing stops on Flare: the moving trigger stays sealed inside FCC until execution, then settlement is checked against FTSO.
```

## Bounty / track selection

```
Bounty 2 — Confidential Compute Apps
```

## Description

```
On-chain stop-loss orders leak the one thing they must protect: the trigger price. It sits in public contract storage or a keeper's mempool, so anyone can read every trader's liquidation level, push the price to it, and absorb the forced sell. Centralized exchanges hide your stops; DeFi today cannot.

DarkStop fixes this with Flare Confidential Compute. A fixed trigger or trailing distance is ECIES-encrypted in the browser to the TEE extension's enclave key. On-chain, placeOrder(ciphertext) stores only the deposit and an opaque blob. For a trailing stop, the Go extension privately updates a high-watermark from fresh FTSO samples and moves the hidden trigger with it; neither value appears in placement calldata or contract storage. When the moving boundary is crossed, the extension submits settle(), revealing the effective trigger. The vault independently enforces executor authority and re-reads a fresh FTSO price at-or-below that trigger before a test USDT0 payout. In this prototype, encrypted-policy integrity still relies on the authorized TEE.

What works today: the vault and extension artifacts are deployed and registered on Coston2 (extension id 503, TEE machine registered on-chain); one command runs the complete local simulated-FCC encrypted trailing-policy place → private high-watermark update → real Go watcher settle → enclave/chain consistency proof; Coston2 fork tests exercise settlement against the live FTSO feed; and the browser encryptor is proven wire-compatible with go-ethereum's ECIES by a cross-language conformance suite.

Honesty section: the complete demo loop runs on a local simulated FCC stack. The registered Coston2 TEE uses the scaffold's simulated mode, which the Flare team confirmed in the hackathon Telegram is accepted for judging. The repository's trailing-policy runtime is v0.2.0 and requires a fresh Coston2 version/code-hash registration; live-testnet placeOrder is currently gated on the Flare-side FTDC proxy producing an availability proof for our registered machine (our proxy is provably healthy — Flare's proxy polls its TEE_INFO every ~10s). This is documented and escalated in docs/coston2-runbook.md.
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
FCC (Flare Confidential Compute): DarkStopVault implements the real FCC instruction-sender path. The extension is registered on TeeExtensionRegistry (id 503), instruction fees pass through sendInstructions, OPType/OPCommand (DARKSTOP / PLACE_ORDER / CANCEL_ORDER) are mirrored byte-for-byte across Solidity constants, Go config, and decoder registration, and the TEE machine is registered on-chain. The complete loop is demonstrated on the local simulated FCC stack; the Coston2 availability limitation is disclosed separately. The product's pre-execution confidentiality requires the TEE and cannot be implemented by a public smart contract alone.

FTSO — used in two places:
1. Inside the TEE: the watcher polls the block-latency FLR/USD feed to detect trigger crossings privately, so monitoring leaks nothing.
2. On-chain at settlement: settle() calls FtsoV2.getFeedById(FLR_USD) itself and requires the price to be fresh and at-or-below the revealed trigger. The executor may request a stricter window, but the contract rejects anything above 300 seconds. FTSO is the on-chain arbiter of current price; the authorized TEE remains responsible for encrypted-policy integrity.
```

## What was built during the hackathon (new work statement)

```
Everything in DarkStop was built during the hackathon on top of the official flare-foundation/fce-extension-scaffold (the Hello World FCC template, credited in the README). We built: DarkStopVault.sol (deposit vault + FCC instruction sender + FTSO-verified settlement + cancel/refund), the Go TEE extension logic (ECIES decryption, in-enclave order store, fixed and trailing-stop policies, private high-watermark tracking, PLACE_ORDER/CANCEL_ORDER handlers, FTSO watcher with retry/backoff, non-blocking receipt/vault reconciliation, same-nonce fee bumps, and audit logging), a Next.js frontend whose browser ECIES adapter uses audited Noble primitives and is wire-compatible with go-ethereum (eciesjs is not — verified by a cross-language conformance suite), Coston2 deployment tooling, a one-shot local dev stack with negative guardrail proofs, fork tests against the live Coston2 FTSO, and a Coston2 TEE bring-up runbook. No pre-existing DarkStop product or codebase was reused.
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
3. Multiple trading pairs and encrypted OCO take-profit orders.
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
