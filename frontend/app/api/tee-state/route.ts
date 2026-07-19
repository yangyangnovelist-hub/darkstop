// Proxies the TEE extension's GET /state (enclave pubkey + order list) so the
// browser can fetch it same-origin, dodging CORS on the extension server.
export const dynamic = "force-dynamic";

export async function GET() {
  const url = process.env.TEE_STATE_URL ?? "http://localhost:7702/state";
  try {
    const res = await fetch(url, {
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
    if (!res.ok) throw new Error(`upstream returned ${res.status}`);
    return Response.json(await res.json());
  } catch (err) {
    // Local dev-stack fallback: no TEE extension running, encrypt to the
    // repo's fixture keypair instead (set by scripts/dev-stack.sh).
    const fallback = process.env.DEV_FALLBACK_TEE_PUBKEY;
    if (fallback) {
      return Response.json({
        stateVersion: `0x${"0".repeat(64)}`,
        state: {
          encryptionPubKey: fallback,
          supportedPolicies: ["fixed"],
          openOrders: 0,
          orders: [],
        },
        devFallback: true,
      });
    }
    return Response.json(
      { error: `TEE state unreachable at ${url}: ${String(err)}` },
      { status: 502 },
    );
  }
}
