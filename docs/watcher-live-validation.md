# Watcher live validation on real Coston2

Purpose: prove the refactored FTSO watcher (`internal/extension/watcher.go`)
settles real orders on the **live Coston2 testnet** — not just anvil/mock.

- Date: 2026-07-19
- Chain: Coston2 (chain id 114), RPC `https://coston2-api.flare.network/ext/C/rpc`
- Deployer / executor: `0x995662F9Bdbd561CD58BA665a12Db426CC3c9BD6`
- Explorer: `https://coston2-explorer.flare.network`

## Why a fresh dev-stack deployment

The production vault `0xd93E8F7dE2A5A7C4eC45F115f7047103da2dD8bF` cannot accept
`placeOrder` on Coston2 — it routes through the real Flare FTDC/TEE machine
registry, which reverts (`TooMany`) because no enclave is registered. The
dev-stack (`contracts/dev/DevStack.s.sol`) uses **mock** TEE registries + a
test-settable `MockFtsoV2`, so its `placeOrder` does not touch the real TEE
registry. Deploying that stack to real Coston2 gives a working end-to-end
settlement path against genuine on-chain state. Everything below is real
Coston2 (mock *contracts*, real *chain* — every tx is on the public explorer).

## Deployed contracts (real Coston2)

Deployed with `forge script contracts/dev/DevStack.s.sol:DevStack --broadcast
--legacy` using `.env` `DEPLOYMENT_PRIVATE_KEY`.

| Contract | Address | Deploy tx |
|---|---|---|
| DarkStopVault | `0x1188c97b49080f8455b16F7304a54fC75527699a` | `0x767f2db66a3adc44048a5704c3a4ae3945b48dccf6466bdeb42b6141a13e977e` |
| MockUSDT0 (payout, 6dp) | `0x30b768be11184Aa4E8Ba00664d407B228D437202` | `0xc2f6117dd32a4682ba2cc5085d4a3fa0fdb89de98ff85c514ef03a3511d4beee` |
| MockFtsoV2 | `0x069656e430D3C40b0c210FafFeA562D16947FCB8` | `0xd09f584a2b52d915a4cf42eb10d4d7c3c03942a35bd1d0172e028566435c62f6` |
| MockTeeExtensionRegistry | `0x82E991a6f8e10387dB829a2FBdef9AC41F127c20` | `0x802d4862b4064262c969227a1b932e20bcedd2fe5a60b8ebee99d8de0608e621` |
| MockTeeMachineRegistry | `0xd9dBB1E2898A302b41e059cf1Eb30DE7CFEF9a34` | `0xbbf252784709194aef43b6ebb69bc405841b8d99efe8096daf06be7d3462efad` |

Post-deploy state (verified via `cast call`):
`teeExecutor` = deployer; `INSTRUCTION_FEE` = 0.01 ether; vault USDT0 balance =
`1000000000000` (1,000,000 USDT0 payout pool); FTSO feed seeded to `300000@7dp`
= $0.03.

## 1. Place order on real chain

`frontend/lib/ecies.ts` (`encryptTriggerPrice`) encrypted a fixed **$0.02**
trigger (`20000` @ 6dp) to the extension's ECIES pubkey, then `placeOrder`
(deposit 1 FLR + 0.01 fee) was sent from the deployer:

- placeOrder tx: `0x7805898ede5fc8bfbacc147087414ae975289bd198a848362ce378e1a99af985` (block 33018342, status success)
- `OrderPlaced` → orderId **1**, owner `0x995662…9BD6`
- Instruction relayed to extension `/action`; TEE decrypted + stored order 1.
  `/state` then showed `openOrders:1, orders:[{orderId:"1",status:"open"}]`
  (**no trigger price exposed** — privacy preserved).

## 2. Watcher running against real Coston2

`cmd/main.go` started with `CHAIN_URL=<Coston2>`,
`VAULT_ADDRESS=0x1188c97b…699a`, `EXECUTOR_PRIVATE_KEY=<deployer>`. It:

- resolved FtsoV2 from the vault immutable on the live chain,
- polled FLR/USD every 2s, reporting `= 30000 (6dp)` while price stayed above
  the trigger and holding order 1 open (no settle).

## 3. Price drop → real settlement

Dropped the feed to `100000@7dp` = **$0.01** (below the $0.02 trigger) via
`cast send MockFtsoV2.setFeed` (tx `0xdbd99004a9ffeb7d97b5cb93e507c70b8603b332360c64c949c011d6a47799ca`).

Watcher log (real chain):

```
watcher: FLR/USD = 10000 (6dp, feed age 5s), 1 open order(s)
watcher: order 1 TRIGGERED (price 10000 <= trigger 20000) — settling
watcher: order 1 settle attempt 1/3
watcher: order 1 settle tx sent: 0xb0d158681c53564f4265218d941d0ea2be99c576e4977aade1f4b5a2ffced2cb (nonce 32, gas 86570, gasPrice 650000000000)
watcher: order 1 tx 0xb0d158…d2cb has an unresolved broadcast/receipt — reconciling on later ticks
watcher: order 1 is executed on-chain — local state reconciled without receipt
watcher: FLR/USD = 10000 (6dp, feed age 11s), 0 open order(s)
```

### Settle transaction (real, explorer-verifiable)

- **Settle tx: `0xb0d158681c53564f4265218d941d0ea2be99c576e4977aade1f4b5a2ffced2cb`**
  - `https://coston2-explorer.flare.network/tx/0xb0d158681c53564f4265218d941d0ea2be99c576e4977aade1f4b5a2ffced2cb`
  - block 33018361, status **success**, from executor → vault, gasUsed 85647, nonce 32
  - `OrderExecuted(1, 10000)` emitted (price `0x2710` = 10000 = $0.01 @ 6dp)
  - ERC-20 `Transfer` of `10000` USDT0 to the order owner (real payout:
    1 FLR × $0.01 = $0.01)
- On-chain `orders(1)` status = **2 (EXECUTED)**; owner USDT0 balance = `10000`.
- Extension `/state` after: `orders:[{orderId:"1",status:"executed"}]`.

## Real-chain behavior the mock never exercised

On anvil the settle receipt is available synchronously, so `settleOnce`
confirms in-line and the pending map is never used. On **real Coston2** the
receipt was **not** available at the immediate lookup, so the new code:

1. wrapped the broadcast in a `pendingTransactionError` and tracked it in the
   `pending` map (nonce 32) instead of blocking or falsely failing;
2. on the next tick, `reconcilePending` → `TransactionByHash` showed the tx no
   longer pending (mined), then `reconcileAgainstVault` read the canonical
   vault status (`EXECUTED`) and reconciled local state **without a receipt**,
   correctly releasing the consumed nonce.

This is exactly the pending-tracking / nonce-reconciliation / reconcile-against-
vault path the refactor added, and it worked on the first live order. No
"already known", nonce lag, or dropped-tx anomaly occurred in this run; the
receipt-lag path (the main real-vs-mock difference) was hit naturally and
handled correctly.

**No watcher bug was exposed — no source change was needed.** The refactored
watcher settles real orders on real Coston2.

## Regression suite (still green after live run)

- `go test ./...` — pass
- `forge test` — 21 passed
- frontend `vitest` — 25 passed
