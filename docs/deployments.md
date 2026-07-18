# DarkStop deployments

## Coston2 — 2026-07-16 (Task 2.2 / Phase 4 toolchain)

Deployed by `./scripts/pre-build.sh` (deployer = `0x995662F9Bdbd561CD58BA665a12Db426CC3c9BD6`, chain id 114).

| Contract / item | Address / value | Tx |
|---|---|---|
| MockUSDT0 (payout token, 6 decimals) | `0x6196b20FaeCE88ace220297122bB170A5B97b60F` | `0xe8e235dd9d9d29669bb0a7f85eedcebfebb422ae38d067de78ceafb297e70186` |
| DarkStopVault (instruction sender) | `0xd93E8F7dE2A5A7C4eC45F115f7047103da2dD8bF` | `0xf444c5c2d6598d3ac889fafbe305f38e11b9424ac36140b02eb419e38355f9c0` |
| Payout pool mint (1,000,000 USDT0 = 1e12 base units → vault) | — | `0x5bfb1102c17dfc9c02883b011b4c51ab574151180f91130f10502091f4a50eb6` |
| `setTeeExecutor(deployer)` (testnet: executor = deployer) | — | `0x7ea139e537075f077901fd32a7ae57b625ec39524b6d2839024332c8f8c351d2` |
| Extension ID (TeeExtensionRegistry) | `0x…01f7` (= 503) | registration txs: `0x206d90f7…`, `0xb8531d10…`, `0xd7145e18…`, `0x1362bdad…` (see explorer) |
| `setExtensionId()` on vault | — | `0x3a475c71d52276c8cdc3ff2d0d9ffba2f1ae40ca0fd5dd666a3a48f5938d38b2` |
| FtsoV2 (resolved on-chain via FlareContractRegistry `getContractAddressByName("FtsoV2")` — same address `getTestFtsoV2()` resolves) | `0xC4e9c78EA53db782E28f28Fdf80BaF59336B304d` | — |
| INSTRUCTION_FEE | `1000000` wei (registry min fee is 1000 wei/TEE, `FeeTooLow` enforces a floor — overpay accepted) | — |

Explorer: `https://coston2-explorer.flare.network/address/0xd93E8F7dE2A5A7C4eC45F115f7047103da2dD8bF`

## On-chain smoke test (2026-07-16)

```bash
RPC=https://coston2-api.flare.network/ext/C/rpc
VAULT=0xd93E8F7dE2A5A7C4eC45F115f7047103da2dD8bF
USDT0=0x6196b20FaeCE88ace220297122bB170A5B97b60F

cast call $VAULT 'FTSO_V2()(address)'      --rpc-url $RPC  # 0xC4e9c78E…B304d ✓ non-zero
cast call $VAULT 'PAYOUT_TOKEN()(address)' --rpc-url $RPC  # 0x6196b20F…b60F ✓
cast call $VAULT 'teeExecutor()(address)'  --rpc-url $RPC  # 0x995662F9…9BD6 ✓ (= deployer)
cast call $VAULT 'OWNER()(address)'        --rpc-url $RPC  # 0x995662F9…9BD6 ✓
cast call $VAULT 'INSTRUCTION_FEE()(uint256)' --rpc-url $RPC        # 1000000 ✓
cast call $USDT0 'balanceOf(address)(uint256)' $VAULT --rpc-url $RPC # 1000000000000 (= 1,000,000e6) ✓

# setExtensionId (success, tx 0x3a475c71d52276c8cdc3ff2d0d9ffba2f1ae40ca0fd5dd666a3a48f5938d38b2)
cast send $VAULT 'setExtensionId()' --private-key "$DEPLOYMENT_PRIVATE_KEY" --rpc-url $RPC
```

### placeOrder — blocked on TEE machine registration (expected)

```bash
cast estimate $VAULT 'placeOrder(bytes)' 0xabab…(64 bytes) --value 0.5ether \
  --from 0x995662F9Bdbd561CD58BA665a12Db426CC3c9BD6 --rpc-url $RPC
# → execution reverted, selector 0xd65ac61e = TooMany()
```

`placeOrder` → `_sendInstruction` → `TEE_MACHINE_REGISTRY.getRandomTeeIds(503, 1)` reverts
`TooMany()` because no machine for extension 503 has reached the production set. Our
simulated TEE machine completed the on-chain registration and availability-request steps
(`config/register-tee.state` records `completed_steps: "ra"`), but Flare's Coston2 FTDC
proxy still returns HTTP 404 for the resulting availability proof. Flare confirmed that
FCC is being reworked on Coston2 and that this failure is infrastructure-side; see
`docs/coston2-runbook.md` for the full trace and ruling. The same 404 was re-verified on
2026-07-18. No gas was wasted on a mined revert; the estimate was left as the record.
Re-run the placeOrder + cancel smoke only after Flare restores FTDC proof production.

### Compensating verification: Coston2 fork suite (live FTSO, mocked TEE registries)

```bash
forge test --match-contract DarkStopVaultForkTest --fork-url "$CHAIN_URL"
```

4/4 pass at fork head (2026-07-16): `FeedIsLiveAndFresh`, `PlaceAndSettleAgainstLiveFtso`
(placeOrder emits OrderPlaced, settle re-checks the real FLR/USD feed and pays USDT0),
`SettleRevertsWhenPriceAboveTrigger`, `SettleRevertsOnStaleFeed`. FLR/USD at test time:
value 662726, decimals 8 (≈ $0.0066), timestamp fresh.

## Superseded deployments

| Date | Contract | Address | Note |
|---|---|---|---|
| 2026-07-16 (Phase 0 spike) | HelloWorldInstructionSender | `0xE8Dd854cc8f77D98397Ba41b1bd1537976d5c6f0` | hello-spike scratch copy, extension id 502 |
