"use client";

import { useState, type FormEvent } from "react";
import { formatEther, parseEther, parseUnits } from "viem";
import { useAccount, useReadContract, useWriteContract } from "wagmi";
import { chain, orderingEnabled, txUrl } from "@/lib/chain";
import { VAULT_ADDRESS, vaultAbi } from "@/lib/vault";
import { encryptTrailingStop, encryptTriggerPrice } from "@/lib/ecies";
import { percentToBps } from "@/lib/strategy";

type Phase =
  | { kind: "idle" }
  | { kind: "encrypting" }
  | { kind: "signing" }
  | { kind: "sent"; hash: string }
  | { kind: "error"; message: string };

// Turn cryptic wallet/RPC errors into something a person can act on.
function humanizeError(err: unknown): string {
  const raw = err instanceof Error ? err.message : String(err);
  const first = raw.split("\n")[0];
  const lower = raw.toLowerCase();
  if (lower.includes("user rejected") || lower.includes("user denied"))
    return "Transaction cancelled in your wallet.";
  if (lower.includes("insufficient funds"))
    return "Not enough balance for the deposit plus the instruction fee.";
  return first;
}

export function OrderForm() {
  const { isConnected, chainId } = useAccount();
  const { data: fee } = useReadContract({
    address: VAULT_ADDRESS,
    abi: vaultAbi,
    functionName: "INSTRUCTION_FEE",
  });
  const { writeContractAsync } = useWriteContract();

  const [amount, setAmount] = useState("1");
  const [trigger, setTrigger] = useState("0.02");
  const [strategy, setStrategy] = useState<"fixed" | "trailing">("trailing");
  const [trailPercent, setTrailPercent] = useState("5");
  const [phase, setPhase] = useState<Phase>({ kind: "idle" });

  const busy = phase.kind === "encrypting" || phase.kind === "signing";
  const ready = orderingEnabled && isConnected && chainId === chain.id && fee !== undefined;

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!ready || fee === undefined) return;
    try {
      setPhase({ kind: "encrypting" });

      const deposit = parseEther(amount);
      if (deposit <= 0n) throw new Error("Deposit must be positive.");
      const res = await fetch("/api/tee-state");
      const body = await res.json();
      if (!res.ok) throw new Error(body?.error ?? "TEE state unavailable.");
      const pubKey: string | undefined = body?.state?.encryptionPubKey;
      if (!pubKey) throw new Error("TEE state response has no encryption key.");
      const supported: unknown = body?.state?.supportedPolicies;
      if (!Array.isArray(supported) || !supported.includes(strategy)) {
        throw new Error(`Connected TEE does not advertise ${strategy} policy support.`);
      }

      // Encrypted in the browser; only this ciphertext ever goes on chain.
      const ciphertext = strategy === "fixed"
        ? encryptTriggerPrice(pubKey, parseUnits(trigger, 6).toString())
        : encryptTrailingStop(pubKey, percentToBps(trailPercent));

      setPhase({ kind: "signing" });
      const hash = await writeContractAsync({
        address: VAULT_ADDRESS,
        abi: vaultAbi,
        functionName: "placeOrder",
        args: [ciphertext],
        value: deposit + fee,
      });
      setPhase({ kind: "sent", hash });
    } catch (err) {
      setPhase({ kind: "error", message: humanizeError(err) });
    }
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-4">
      <div className="grid grid-cols-2 gap-2 rounded-xl border border-zinc-800 bg-black/30 p-1">
        {(["trailing", "fixed"] as const).map((option) => (
          <button key={option} type="button" onClick={() => setStrategy(option)}
            aria-pressed={strategy === option}
            className={`rounded-lg px-3 py-2 text-xs font-semibold uppercase tracking-wider transition ${strategy === option ? "bg-emerald-400 text-black" : "text-zinc-500 hover:text-zinc-200"}`}>
            {option === "trailing" ? "Private trailing" : "Fixed stop"}
          </button>
        ))}
      </div>
      <label className="flex flex-col gap-1.5">
        <span className="text-sm text-zinc-400">
          Test position deposit ({chain.nativeCurrency.symbol})
        </span>
        <input
          type="number"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          required
          inputMode="decimal"
          pattern="[0-9]*\.?[0-9]+"
          min="0.000000000000000001"
          step="any"
          className="rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 font-mono text-zinc-100 outline-none transition focus:border-emerald-500"
        />
      </label>

      <label className="flex flex-col gap-1.5">
        <span className="text-sm text-zinc-400">{strategy === "fixed" ? "Trigger price (USD per FLR)" : "Private trailing distance (%)"}</span>
        <input
          type="number"
          value={strategy === "fixed" ? trigger : trailPercent}
          onChange={(e) => strategy === "fixed" ? setTrigger(e.target.value) : setTrailPercent(e.target.value)}
          required
          inputMode="decimal"
          pattern="[0-9]*\.?[0-9]+"
          min={strategy === "trailing" ? "0.25" : "0.000001"}
          max={strategy === "trailing" ? "50" : undefined}
          step={strategy === "trailing" ? "0.01" : "0.000001"}
          className="rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 font-mono text-zinc-100 outline-none transition focus:border-emerald-500"
        />
      </label>

      <p className="text-xs text-zinc-500">
        {strategy === "fixed" ? "Executes at your encrypted price." : "The TEE updates a private high-watermark from fresh FTSO samples and moves the hidden trigger with it."} The policy is ECIES-encrypted in your browser before it is sent.
        {fee !== undefined && (
          <>
            {" "}
            Instruction fee: {formatEther(fee)} {chain.nativeCurrency.symbol}{" "}
            (added to your deposit).
          </>
        )}
      </p>

      <button
        type="submit"
        disabled={!ready || busy}
        className="rounded-lg bg-emerald-500 px-4 py-2.5 font-semibold text-black transition hover:bg-emerald-400 disabled:cursor-not-allowed disabled:opacity-40"
      >
        {phase.kind === "encrypting"
          ? "Encrypting…"
          : phase.kind === "signing"
            ? "Confirm in wallet…"
            : "Create encrypted stop"}
      </button>

      {!isConnected && orderingEnabled && (
        <p className="text-xs text-zinc-500">Connect a wallet to place orders.</p>
      )}
      {!orderingEnabled && (
        <p className="text-xs text-amber-400">Live Coston2 ordering is paused; use the working local demo linked above.</p>
      )}
      {phase.kind === "sent" && (
        <p className="text-sm text-emerald-400">
          Order submitted.{" "}
          {txUrl(phase.hash) ? (
            <a
              href={txUrl(phase.hash)!}
              target="_blank"
              rel="noreferrer"
              className="underline decoration-emerald-600 underline-offset-2 hover:text-emerald-300"
            >
              View transaction
            </a>
          ) : (
            <span className="font-mono text-xs">{phase.hash}</span>
          )}
        </p>
      )}
      {phase.kind === "error" && (
        <p className="break-words text-sm text-red-400">{phase.message}</p>
      )}
    </form>
  );
}
