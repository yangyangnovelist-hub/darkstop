# DarkStop — 3-Minute Demo Video Script

Designed to be recorded **without any voiceover**: every scene is screen
action + on-screen captions. Captions are English, max 12 words each — add
them as text overlays (or type them into a large text editor window shown
briefly before each scene). The 操作说明 column is in Chinese for the person
recording; it never appears in the video.

Total target length: ~3:00. Practice each scene once before recording.

## Preparation (before recording — 录制前准备, 不计入视频)

操作说明（中文）:

1. 终端进入仓库根目录：`cd ~/Desktop/hackathons/darkstop`
2. 提前跑一遍本地栈确认一切正常，然后停掉重来，保证录制时是干净状态：
   `./scripts/dev-stack.sh stop`（如有残留）
3. 浏览器准备 4 个标签页（先都打开好，录制时只切换）：
   - Tab 1: `http://localhost:3000`（先别加载，dev server 起来后再刷新）
   - Tab 2: 本地 explorer 不存在——改用 MetaMask 的交易详情 + 终端里
     `cast tx <hash>` 展示 calldata（见 Scene 2）
   - Tab 3: `https://coston2-explorer.flare.network/address/0xd93E8F7dE2A5A7C4eC45F115f7047103da2dD8bF`
   - Tab 4: GitHub 仓库页（提交后的公开 repo）
4. 编辑器打开 `contracts/DarkStopVault.sol`，光标停在 `settle()`（约第 180 行），
   字号调大到 16pt 以上
5. MetaMask 已导入 anvil key #0，网络已添加 `http://127.0.0.1:8545` chain id 31337
6. 隐藏无关的书签栏、通知，屏幕分辨率见附录

---

## Scene 1 — The problem (0:00–0:15)

| | |
|---|---|
| **Screen** | A simple static slide (make it in any tool, or full-screen a text file): line 1 "Your stop-loss order is PUBLIC on-chain." line 2 "Predators read your trigger price and hunt it." Optionally a screenshot of any DEX limit-order contract storage. |
| **Captions** | 1. `On-chain stop orders leak your trigger price.` 2. `Anyone can hunt your liquidation level.` 3. `DarkStop: the trigger price never touches the chain.` |
| **操作说明** | 提前做好一页幻灯片（Keynote 或全屏文本文件都行），录制时静置 15 秒。三条字幕依次叠加显示（每条约 5 秒）。 |

## Scene 2 — Place an encrypted order (0:15–1:10)

| | |
|---|---|
| **Screen** | Terminal: run `./scripts/dev-stack.sh` (output scrolls: anvil starts, vault deployed). Then browser Tab 1 (`localhost:3000`): connect wallet, fill deposit `1` FLR, trigger `0.02` USD, click Place Order, approve in MetaMask. Then back in the terminal: `cast tx <tx-hash> --rpc-url http://127.0.0.1:8545` — **zoom/enlarge on the `input` field**: a long opaque hex blob. Point the cursor at it and hold for 3 seconds. |
| **Captions** | 1. `Start the stack: one script.` 2. `Place a stop-loss. Trigger price: $0.02.` 3. `The trigger is ECIES-encrypted in the browser.` 4. `On-chain calldata: unreadable ciphertext. No price anywhere.` |
| **操作说明** | ① 终端输入 `./scripts/dev-stack.sh` 回车，等它跑完（约 20 秒，输出会打印 VAULT/FTSO 地址，**复制保存这两个地址**，Scene 3 要用）。② 新开终端标签 `cd frontend && npm run dev`，等 ready。③ 切浏览器 Tab 1 刷新，点 Connect Wallet → MetaMask 确认。④ 表单填 deposit `1`、trigger `0.02`，点 Place Order，MetaMask 弹窗点确认。⑤ UI 出现 Pending 订单行后，**复制交易 hash**（MetaMask 活动记录里能看到）。⑥ 切回终端敲 `cast tx <粘贴hash> --rpc-url http://127.0.0.1:8545`，滚动到 `input` 字段，用 macOS 的 ctrl+滚轮 或录屏后期放大该区域，停留 3 秒。 |

## Scene 3 — Price crosses, order auto-executes (1:10–1:50)

| | |
|---|---|
| **Screen** | Split view or quick switch: terminal + browser Tab 1 side by side if possible. Terminal: run the two `cast send` commands (price drop, then settle). Browser: the order row flips **Pending → Executed** within ~2 seconds. Hold on the flipped row. |
| **Captions** | 1. `FLR/USD drops below the trigger.` 2. `The TEE reveals the trigger and settles on-chain.` 3. `Pending → Executed. Payout in USDT0. No human touched it.` |
| **操作说明** | 用 Scene 2 保存的两个地址替换 `$FTSO` 和 `$VAULT`。① 模拟价格跌破触发价：`cast send $FTSO 'setFeed(uint256,int8,uint64)' 150000 7 $(date +%s) --rpc-url http://127.0.0.1:8545 --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80` ② 模拟 TEE 结算：`cast send $VAULT 'settle(uint256,uint256,uint256)' 1 20000 300 --rpc-url http://127.0.0.1:8545 --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80` ③ 立刻切到浏览器（最好提前把浏览器和终端左右分屏），等约 2 秒，订单行从 Pending 翻成 Executed，镜头停 4 秒。注意：这两条命令模拟的是本地栈里 TEE watcher 的动作；真实 TEE extension 的 watcher 有 77 个测试覆盖。 |

## Scene 4 — Real Coston2 deployment + fork tests (1:50–2:25)

| | |
|---|---|
| **Screen** | Browser Tab 3: Coston2 explorer page of the vault (`0xd93E…D8bF`) — scroll slowly past the contract header and transactions. Then terminal: run the fork test command; end state = green `4 passed` against the live FTSO. Then `forge test` summary (`20 passed`) and `go test ./...` (`ok` lines). |
| **Captions** | 1. `The vault is live on Flare Coston2 testnet.` 2. `Fork tests settle against the REAL FTSO feed.` 3. `20 unit + 4 fork + 77 Go tests. All green.` |
| **操作说明** | ① 切浏览器 Tab 3，缓慢滚动合约页 5 秒。② 切终端，敲：`forge test --match-contract DarkStopVaultForkTest --fork-url https://coston2-api.flare.network/ext/C/rpc`，等绿色 `4 passed`（约 30 秒，可后期剪快）。③ 再敲 `forge test`（很快，20 passed）。④ 再敲 `go test ./...`，出现整列 `ok`。每个绿色结果停 2-3 秒。 |

## Scene 5 — The code that keeps the TEE honest (2:25–2:45)

| | |
|---|---|
| **Screen** | Editor, `contracts/DarkStopVault.sol`, `settle()` — zoom on exactly these three lines: `getFeedById(FLR_USD)` / `require(... "stale price")` / `require(price <= _triggerPrice, ...)`. Highlight/select them. |
| **Captions** | 1. `settle() never trusts the TEE alone.` 2. `It re-reads FTSO on-chain: fresh price, at-or-below trigger.` |
| **操作说明** | 编辑器已提前打开到 `settle()`（约 180-189 行）。用鼠标选中 `getFeedById` 那一行和下面两个 `require` 行（第 185、186、189 行），停 6 秒。字号要大，确保 1080p 下清晰可读。 |

## Scene 6 — Roadmap + close (2:45–3:00)

| | |
|---|---|
| **Screen** | One static slide: project name + logo text "DarkStop", then 4 roadmap bullets, then repo URL. |
| **Captions** (as slide content, not overlays) | Title: `DarkStop — stop-losses that can't be hunted.` Bullets: `Real attested TEE on Songbird` / `Live Coston2 placeOrder (pending Flare FTDC proxy)` / `More pairs, take-profit orders` / `Real DEX settlement`. Footer: repo URL + `Built on Flare FCC + FTSO.` |
| **操作说明** | 提前做好结尾幻灯片，静置 15 秒结束录制。 |

---

## Appendix A — Caption placement (字幕怎么加)

最简单的两种方式，选其一：

1. **iMovie（推荐）**: 录完屏后拖进 iMovie，用 Titles → Lower Third 逐条加英文字幕，导出 1080p。
2. **录制时叠加**: 提前把每条字幕放进一个置顶的小窗口（如 macOS 便笺 Stickies 置顶），录制时手动切换显示。效果略糙但零剪辑。

## Appendix B — Recording tool & resolution (录屏工具与分辨率)

- **工具**: macOS 自带 **Screenshot 工具**（按 `Cmd+Shift+5` → 选 Record Entire Screen 或 Record Selected Portion）或 **QuickTime Player**（File → New Screen Recording）。两者都免费且无水印。
- **分辨率**: 用 **1920×1080** 输出。如果是 Retina 屏，录制选定区域设为 1920×1080 的窗口区域，或录全屏后在 iMovie 导出时选 1080p。
- **字号**: 终端和编辑器字号 ≥16pt；浏览器缩放 125%。录完先自己在手机上看一遍，确认所有文字可读。
- **鼠标**: `Cmd+Shift+5` 的 Options 里勾选 Show Mouse Clicks。
- **时长**: DoraHacks 无严格上限，但评委注意力有限——**控制在 3 分钟内**。超了就剪掉等待过程（测试跑动、dev-stack 部署）用 2× 快放。
- **上传**: 导出 mp4 后传 YouTube（Unlisted 即可），把链接填进提交表单（见 `docs/submission-form.md`）。
