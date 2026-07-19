# DarkStop — Championship Demo Script

Target runtime: **1:45–2:00**. Record the product first; use terminal and code
only as short evidence. All captions are English and describe shipped behavior.

## Preparation

1. Run `./scripts/demo-e2e.sh` once before recording to verify the machine.
2. Start a clean stack: `./scripts/dev-stack.sh stop && ./scripts/dev-stack.sh`.
3. Start the frontend: `cd frontend && npm run dev`.
4. Open the local DarkStop app, the separate Coston2 vault explorer, and the repository tests.
5. Use a 1920×1080 canvas, browser zoom 110–125%, and hide bookmarks.

## 0:00–0:18 — Show the attack surface

**Screen:** DarkStop hero and the side-by-side observer comparison.

**Captions:**

- `Public stop orders reveal exactly where traders will sell.`
- `DarkStop puts only ciphertext in the placement transaction.`
- `Before execution, a chain observer cannot see the strategy.`

Keep the cursor on the public `$0.020000 · exposed` value, then move to the
sealed `0x04…` value. Do not begin with a title slide.

## 0:18–0:48 — Create a private trailing stop

**Screen:** Order form. Select **Private trailing**, enter `5%`, deposit `1 test FLR`,
and submit. If wallet interaction is unreliable during recording, run
`cd frontend && npx tsx scripts/place-order.ts` in a narrow terminal beside the
app; the script uses the same browser-compatible ECIES implementation.

**Captions:**

- `A 5% trailing stop updates from fresh FTSO samples.`
- `The high-watermark and moving trigger exist only inside FCC.`
- `The transaction carries one opaque encrypted policy.`

Show the order appear as **Pending**. Briefly show the real local placement
transaction hash and its ciphertext input, never a private key or `.env` file.

## 0:48–1:15 — Prove hidden state changes

**Screen:** A compact terminal beside the live order table. Run
`./scripts/demo-settle.sh` after the watcher has observed the initial price.

**Captions:**

- `FTSO establishes the private high-watermark.`
- `Price falls through the moving 5% boundary.`
- `The Go watcher triggers automatically. No manual settle call.`

Before the price drop, briefly run `./scripts/demo-guardrails.sh`. It must show
two green **BLOCKED** lines: an outsider cannot settle, and even the configured
executor cannot settle before the FTSO condition. Then run
`./scripts/demo-settle.sh`. The money shot is the row changing
**Pending → Executed** without clicking a settle button.

## 1:15–1:36 — Verify the receipt

**Screen:** Hold on the expanded execution proof under the completed order,
then open the execution transaction.

**Captions:**

- `The vault accepts only its configured TEE executor.`
- `It re-reads a fresh FTSO price on-chain.`
- `The test USDT0 payout and receipt are real contract state changes.`

Say or caption this disclosure exactly once:

`This full loop is local simulated FCC. The deployed Coston2 artifacts and live-FTSO fork evidence are separate.`

## 1:36–1:52 — Engineering evidence

**Screen:** Fast cuts of green results only:

- `forge test`
- `go test ./...`
- `npm test`

**Caption:** `Solidity, Go, browser crypto, and cross-language vectors tested.`

Do not scroll through source code and do not narrate test names.

## 1:52–2:00 — Close on the product

**Screen:** Return to the DarkStop hero.

**Final caption:**

`DarkStop — strategies that execute without announcing themselves.`

Footer: `Local simulated FCC loop · Coston2 artifacts + live FTSO fork · github.com/yangyangnovelist-hub/darkstop`

## Recording gate

- The complete flow must work twice before recording.
- No roadmap slide.
- No claim of mainnet or hardware-attested execution.
- Never present the illustrative privacy comparison as a live Coston2 order.
- Call the payout `test USDT0`, not a DEX swap.
- Never expose `.env`, private keys, Telegram exports, or indexer credentials.
