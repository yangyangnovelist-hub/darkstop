import { ConnectWallet } from "@/components/ConnectWallet";
import { OrderForm } from "@/components/OrderForm";
import { OrdersTable } from "@/components/OrdersTable";
import { addressUrl } from "@/lib/chain";
import { VAULT_ADDRESS } from "@/lib/vault";

export default function Home() {
  const vaultUrl = addressUrl(VAULT_ADDRESS);

  return (
    <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-8 px-6 py-10">
      <header className="flex items-center justify-between gap-4">
        <div className="flex items-baseline gap-3">
          <h1 className="text-2xl font-bold tracking-tight">
            Dark<span className="text-emerald-400">Stop</span>
          </h1>
          <span className="hidden text-sm text-zinc-500 sm:inline">
            confidential stop-loss on Flare
          </span>
        </div>
        <ConnectWallet />
      </header>

      <section className="rounded-2xl border border-emerald-500/20 bg-emerald-500/5 px-6 py-5">
        <p className="text-lg font-medium text-emerald-300">
          Your trigger price never touches the chain —{" "}
          {vaultUrl ? (
            <a
              href={vaultUrl}
              target="_blank"
              rel="noreferrer"
              className="underline decoration-emerald-500/60 underline-offset-4 hover:text-emerald-200"
            >
              inspect the calldata yourself
            </a>
          ) : (
            "inspect the calldata yourself"
          )}
          .
        </p>
        <p className="mt-1 text-sm text-zinc-400">
          Orders carry only an ECIES ciphertext, decrypted inside a TEE. At
          settlement the vault re-verifies the live FTSO FLR/USD feed on chain
          — the TEE is never trusted with your funds alone.
        </p>
      </section>

      <div className="grid gap-6 lg:grid-cols-[380px_1fr]">
        <section className="h-fit rounded-2xl border border-zinc-800 bg-zinc-950 p-6">
          <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-zinc-400">
            New stop-loss order
          </h2>
          <OrderForm />
        </section>

        <section className="rounded-2xl border border-zinc-800 bg-zinc-950 p-6">
          <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-zinc-400">
            Orders (live from vault events)
          </h2>
          <OrdersTable />
        </section>
      </div>

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
