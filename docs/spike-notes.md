# Phase 0 spike notes — hello-world scaffold on Coston2

Date: 2026-07-16. Scratch copy at `~/Desktop/hackathons/hello-spike` (not committed).

## What passed

- Faucet funded dev key `0x995662F9Bdbd561CD58BA665a12Db426CC3c9BD6`: 100 C2FLR + 10 USDT0 + 10 FTestXRP (user clicked faucet; captcha blocks automation).
- `./scripts/generate-bindings.sh` — required first; `autogen.go` is gitignored so fresh copies fail pre-flight with `undefined: helloworld.*` until bindings are generated.
- `./scripts/pre-build.sh` against Coston2 (chain id 114):
  - Deployed `HelloWorldInstructionSender` at `0xE8Dd854cc8f77D98397Ba41b1bd1537976d5c6f0`
  - Registered extension on `TeeExtensionRegistry`, Extension ID `0x...01f6` (= 502)
  - Wrote `config/extension.env`
  - FlareTeeManager on Coston2: `0x004224fa1BF1Acd3D233f011FB03b8dd5fA5d41F`

## Gotchas found

- `.env.example`'s `EXT_PROXY_URL=https://<your-ngrok-url>...` breaks `source` (angle brackets) — must be quoted or replaced before any script runs.
- Every fresh clone needs `./scripts/generate-bindings.sh` before `pre-build.sh` (pre-flight compiles tools/).

## Still blocked (waiting on Flare Telegram reply — user asked)

- `docker compose -f docker-compose.yaml -f docker-compose.coston2.yaml up` — ext-proxy needs indexer DB credentials in `config/proxy/extension_proxy.coston2.docker.toml`.
- `post-build.sh` (TEE version/machine registration) and `test.sh` (end-to-end instruction) depend on the proxy + a live ngrok URL (`ngrok http 6674`, then set `EXT_PROXY_URL`).
- Also asked in Telegram: whether judges accept `SIMULATED_TEE=true` demos.
