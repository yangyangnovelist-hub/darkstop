# Coston2 TEE proxy + registration runbook

How to bring the DarkStop extension up on Coston2 end-to-end. Captures the real
sequence and every fix found on 2026-07-16. Secrets (DB password, private keys)
live only in gitignored files — this doc names the fields, not the values.

## Key discovery: skip Docker for the proxy

The `local/tee-proxy` Docker image is NOT public (`registry.gitlab.com/flarenetwork/tee/e2e`
→ 403; building from source needs the private `tee-proxy` + `tee-node` repos). BUT the
`tee-proxy` source is already present at `~/Desktop/tee-proxy` (a go.mod replace target),
so **`start-services.sh --local` builds `start-proxy`/`start-tee` from the scaffold's own
`tools/cmd/` and runs them as Go processes** — only Redis uses Docker. This sidesteps the
image blocker entirely. Use `--local` for Coston2.

## One-time config (gitignored files)

1. `config/proxy/extension_proxy.coston2.toml` (the NON-docker one — `--local` reads this):
   - `[db]` host `34.38.42.208`, port `3306`, database `indexer`, username/password =
     Flare hackathon read-only creds (from Telegram).
   - `redis_port = "127.0.0.1:6382"` (was `:6380` in the example — wrong; compose Redis
     publishes host `6382`, and the host must be explicit or the proxy dials `:6382` with no
     host → "can't assign requested address").
   - `private_key_variable = "PROXY_PRIVATE_KEY"` (was `""` → defaulted to `PRIVATE_KEY`,
     which isn't in `.env` → panic "no PRIVATE_KEY env variable stored").
2. `.env`: `EXT_PROXY_URL` = the ngrok HTTPS URL (see below); `PROXY_PRIVATE_KEY`,
   `DEPLOYMENT_PRIVATE_KEY` = funded Coston2 dev key; `NORMAL_PROXY_URL=https://tee-proxy-coston2-1.flare.rocks`.

## Bring-up sequence

```bash
export PATH="$PATH:$HOME/.foundry/bin:$(go env GOPATH)/bin"
cd ~/Desktop/hackathons/darkstop

# 1. Docker daemon (Redis only). `open -a Docker` sometimes needs a second launch —
#    poll `docker ps` until it responds.
open -a Docker

# 2. ngrok → the proxy's EXTERNAL port. In --local mode that is 6664 (NOT 6674 from the
#    README, which is the docker-mode mapping). Point ngrok at 6664 or /info 404s.
ngrok http 6664           # copy https URL into EXT_PROXY_URL in .env
docker network create docker_default 2>/dev/null || true   # --local starts Redis via base compose which declares this external net

# 3. Start proxy + TEE as Go processes, reading the coston2 toml with creds:
export PROXY_CONFIG="$PWD/config/proxy/extension_proxy.coston2.toml"
CHAIN=coston2 PROXY_CONFIG="$PROXY_CONFIG" ./scripts/start-services.sh --local --chain coston2
curl -sf http://localhost:6664/info    # expect teeInfo{challenge, publicKey...}
curl -sf "$EXT_PROXY_URL/info"         # same, through ngrok — MUST work before post-build

# 4. Register TEE machine on-chain (allow-tee-version + register-tee):
source .env; export EXT_PROXY_URL NORMAL_PROXY_URL DEPLOYMENT_PRIVATE_KEY PROXY_PRIVATE_KEY
./scripts/post-build.sh
```

## Current blocker (2026-07-16, being worked)

`register-tee` progresses: on-chain register (step "r") + availability-check request
(step "a") succeed and are saved to `config/register-tee.state` (`completed_steps: "ra"`).
The final step "p" — `GetFTDCAvailabilityCheckResult` polling the normal/FTDC proxy
(`tee-proxy-coston2-1.flare.rocks`) — first 404s (proof not produced yet; the FTDC proxy
runs ~90s voting rounds and `ActionResult` is a single shot, no retry loop), then after the
proof appears fails with `could not parse token, token is malformed: invalid number of
segments` (a JWT parse error while decoding the FTDC proof). Resume without re-sending:

```bash
cd tools && go run ./cmd/register-tee -a ../config/coston2/deployed-addresses.json \
  -c https://coston2-api.flare.network/ext/C/rpc -p "$EXT_PROXY_URL" -h "$EXT_PROXY_URL" \
  -ep "$NORMAL_PROXY_URL" -state ../config/register-tee.state -resume
```

Suspects for the JWT error: (a) ngrok free-tier browser-interstitial HTML contaminating a
response body that's then parsed as a token — mitigate with the `ngrok-skip-browser-warning`
header or a paid/static domain; (b) a genuine decode mismatch in the scaffold's FTDC proof
handling vs the live Coston2 proxy. Once "p" completes, `nextOrderId`/`placeOrder` unblocks
(the `TooMany()` revert is gone) and `./scripts/test.sh` runs the full PLACE_ORDER round-trip.

## Judge ruling (2026-07-16, Flare Telegram)

Kristaps Grinbergs (Flare Network, admin) confirmed in the hackathon Telegram:
**"We will accept Coston2 simulated approach, no worries."** → the simulated-TEE +
real-Coston2 architecture is officially sanctioned for judging. DB credentials are the
public ones in the group's pinned message (the shared read-only creds), which is what we use.

## RESOLVED root cause + remaining Flare-side blocker (2026-07-16, final)

**JWT error — SOLVED.** `could not parse token, token is malformed` was NOT ngrok/timing/decode.
`register-tee` must run with `SIMULATED_TEE=true` in its OWN env: in simulated mode the TEE's
attestation is the literal `"magic_pass"`, and `GetCodeHashAndPlatform` only skips JWT parsing
when that var is set. Manual `-resume` was run without it. Fix: `export SIMULATED_TEE=true`.

With that fixed, registration reaches the final `ToProduction` ("p") step cleanly. The machine
`0x619E…63f8` IS registered on-chain (host = ngrok URL); step "a" (availability-check request)
lands on-chain (instruction `0xd6aefde2…`).

**Remaining blocker is Flare-side, not ours.** `GetFTDCAvailabilityCheckResult` polls
`https://tee-proxy-coston2-1.flare.rocks` for the availability proof and gets a **consistent 404
across 5+ voting rounds (10+ min)**. Direct `curl .../getActionResult?id=0xd6aefde2…` → `404 page
not found`. Meanwhile our proxy is provably healthy: flare.rocks pulls our TEE_INFO every ~10s
(status 1) — the callback path works. So the FTDC/normal proxy simply is not producing the
availability-check attestation for our instruction. This gates `getRandomTeeIds(503,1)` →
`placeOrder` reverts `TooMany()` on the REAL Coston2 vault.

**Strategic resolution.** Flare admin already ruled the Coston2 *simulated* approach acceptable
for judging. The demo therefore runs on: (1) the local dev-stack (anvil + mock FTSO) which does
full place→settle→execute in the browser — already verified; (2) the real Coston2 deployed
contracts + fork tests proving settle against live FTSO; (3) the live TEE extension processing
PLACE_ORDER instructions locally. The FTDC production-set registration is a Flare-infra formality
that is not blocking the demo. Escalated to Flare Telegram (question drafted) rather than hammered.

## CONFIRMED by Flare (2026-07-16, Kristaps in Telegram)

Direct quote: "We are updating FCC for the Coston2 test network and need to rework the
guides. That is why it is failing atm. Stay tuned for updates. FCC is still in development
and could change before the final release on Flare mainnet." (t.me/c/2376654368/1245)

→ Our diagnosis is officially confirmed: the FTDC availability-proof 404 is Flare rebuilding
FCC on Coston2, NOT our code. Action: stop attempting TEE-machine registration until Flare
ships the update; our extension + proxy are correct and were validated end-to-end up to the
Flare-controlled step. Cite this quote in the README as evidence the on-chain gap is external.
