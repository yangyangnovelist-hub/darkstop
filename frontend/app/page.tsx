import { ConnectWallet } from "@/components/ConnectWallet";
import { OrderForm } from "@/components/OrderForm";
import { OrdersTable } from "@/components/OrdersTable";
import { addressUrl, chain, orderingEnabled } from "@/lib/chain";
import { VAULT_ADDRESS } from "@/lib/vault";

export default function Home() {
  const vaultUrl = addressUrl(VAULT_ADDRESS);

  return (
    <main className="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-7 px-5 py-6 lg:px-8">
      <header className="flex items-center justify-between gap-4">
        <div className="flex items-baseline gap-3">
          <h1 className="font-display text-2xl font-black tracking-[-0.04em]">
            Dark<span className="text-emerald-400">Stop</span>
          </h1>
          <span className="hidden text-sm text-zinc-500 sm:inline">
            confidential stop-loss on Flare
          </span>
        </div>
        <ConnectWallet />
      </header>

      <section className="hero-grid relative overflow-hidden rounded-[2rem] border border-emerald-400/20 bg-[#0b100e] px-6 py-7 lg:px-10 lg:py-9">
        <div className="relative z-10 max-w-3xl">
          <p className="mb-3 font-mono text-[11px] uppercase tracking-[0.28em] text-emerald-400">Confidential execution · {chain.name}</p>
          <h2 className="font-display text-[2.6rem] font-black leading-[0.96] tracking-[-0.055em] text-white sm:text-5xl lg:text-6xl">
            Stops that move.<br/><span className="text-emerald-400">Triggers sealed until execution.</span>
          </h2>
          <p className="mt-4 max-w-2xl text-base leading-7 text-zinc-400">A private trailing-stop engine. Your strategy tracks an FTSO high-watermark inside Flare Confidential Compute, while FTSO independently verifies the settlement price on-chain.</p>
          <div className="mt-6 flex flex-wrap gap-3">
            {orderingEnabled ? (
              <a href="#protect" className="rounded-lg bg-emerald-400 px-4 py-2.5 text-sm font-bold text-black transition hover:bg-emerald-300">Create a private stop</a>
            ) : (
              <a href="https://github.com/yangyangnovelist-hub/darkstop#run-it-yourself" target="_blank" rel="noreferrer" className="rounded-lg bg-emerald-400 px-4 py-2.5 text-sm font-bold text-black transition hover:bg-emerald-300">Run the working local demo ↗</a>
            )}
            {vaultUrl && (
              <a href={vaultUrl} target="_blank" rel="noreferrer" className="rounded-lg border border-zinc-700 bg-black/20 px-4 py-2.5 text-sm font-medium text-zinc-300 transition hover:border-zinc-500 hover:text-white">Inspect the vault ↗</a>
            )}
          </div>
        </div>
        <div className="relative z-10 mt-6 grid grid-cols-3 gap-2 sm:gap-3">
          {[['01','Sealed placement','No trigger in placeOrder'],['02','Private state','High-watermark stays in TEE'],['03','Verified settlement','Fresh FTSO check on-chain']].map(([n,title,copy]) => (
            <div key={n} className="proof-card min-w-0 rounded-xl border border-white/8 bg-black/25 p-3 sm:rounded-2xl sm:p-4">
              <span className="font-mono text-[10px] text-emerald-400 sm:text-xs">{n}</span><p className="mt-3 text-xs font-semibold leading-4 text-white sm:mt-4 sm:text-sm">{title}</p><p className="mt-1 hidden text-xs leading-5 text-zinc-500 sm:block">{copy}</p>
            </div>
          ))}
        </div>
      </section>

      {!orderingEnabled && (
        <section className="rounded-2xl border border-amber-500/20 bg-amber-500/[0.055] px-5 py-4 text-sm text-amber-100">
          <strong>Coston2 artifacts are deployed; new ordering is temporarily unavailable.</strong>{" "}
          Flare&apos;s FTDC availability proof and the v0.2 TEE code-hash registration are still pending. The complete encrypted order loop runs in the linked local simulated-FCC demo.
        </section>
      )}

      <div id="protect" className="grid min-w-0 scroll-mt-6 gap-5 lg:grid-cols-[400px_minmax(0,1fr)]">
        <section className="min-w-0 h-fit rounded-3xl border border-zinc-800 bg-zinc-950/80 p-6 shadow-2xl shadow-black/20">
          <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-zinc-400">
            {orderingEnabled ? "New testnet order" : "Local demo order"}
          </h2>
          <OrderForm />
        </section>

        <section className="min-w-0 rounded-3xl border border-zinc-800 bg-zinc-950/80 p-6 shadow-2xl shadow-black/20">
          <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-zinc-400">
            Orders
          </h2>
          <OrdersTable />
        </section>
      </div>

      <section className="grid gap-3 sm:grid-cols-2">
        <div className="rounded-2xl border border-red-500/15 bg-red-500/[0.035] p-5">
          <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-red-400">Threat-model example</p>
          <p className="mt-3 text-sm text-zinc-300">Public trigger <span className="float-right font-mono text-red-300">$0.020000 · exposed</span></p>
        </div>
        <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/[0.045] p-5">
          <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-emerald-400">Illustrative DarkStop placement</p>
          <p className="mt-3 text-sm text-zinc-300">Policy payload <span className="float-right font-mono text-emerald-300">0x04a7…9f2c · sealed</span></p>
        </div>
      </section>

      <section className="rounded-2xl border border-emerald-500/20 bg-emerald-500/5 px-6 py-5">
        <p className="text-sm font-medium text-emerald-300">
          Placement privacy, stated precisely.{" "}
          {vaultUrl ? (
            <a
              href={vaultUrl}
              target="_blank"
              rel="noreferrer"
              className="underline decoration-emerald-500/60 underline-offset-4 hover:text-emerald-200"
            >
              View the Coston2 contract
            </a>
          ) : (
            "Inspect the contract interface"
          )}
          .
        </p>
        <p className="mt-1 text-sm text-zinc-400">
          <code>placeOrder</code> carries only ECIES ciphertext. The effective
          trigger is disclosed at settlement, when the vault enforces executor
          authority and checks a fresh FTSO FLR/USD price. The current prototype
          relies on the authorized TEE for encrypted-policy integrity.
        </p>
      </section>

      <footer className="mt-auto pt-6 text-xs text-zinc-600">
        Vault: <span className="font-mono">{VAULT_ADDRESS}</span>
        {vaultUrl && (
          <>
            {" · "}
            <a
              href={vaultUrl}
              target="_blank"
              rel="noreferrer"
              className="underline underline-offset-2 hover:text-zinc-400"
            >
              explorer
            </a>
          </>
        )}
      </footer>
    </main>
  );
}
