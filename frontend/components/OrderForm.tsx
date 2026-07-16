"use client";

import { useState, type FormEvent } from "react";
import { parseEther, parseUnits } from "viem";
import { useAccount, useReadContract, useWriteContract } from "wagmi";
import { chain, txUrl } from "@/lib/chain";
import { VAULT_ADDRESS, vaultAbi } from "@/lib/vault";
import { encryptTriggerPrice } from "@/lib/ecies";

type Phase =
  | { kind: "idle" }
  | { kind: "encrypting" }
  | { kind: "signing" }
  | { kind: "sent"; hash: string }
  | { kind: "error"; message: string };

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
  const [phase, setPhase] = useState<Phase>({ kind: "idle" });

  const busy = phase.kind === "encrypting" || phase.kind === "signing";
  const ready = isConnected && chainId === chain.id && fee !== undefined;

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!ready || fee === undefined) return;
    try {
      setPhase({ kind: "encrypting" });

      const deposit = parseEther(amount);
      if (deposit <= 0n) throw new Error("Deposit must be positive.");
      const trigger6 = parseUnits(trigger, 6); // USD/FLR → 6-decimals integer
      if (trigger6 <= 0n) throw new Error("Trigger price must be positive.");

      const res = await fetch("/api/tee-state");
      const body = await res.json();
      if (!res.ok) throw new Error(body?.error ?? "TEE state unavailable.");
      const pubKey: string | undefined = body?.state?.encryptionPubKey;
      if (!pubKey) throw new Error("TEE state response has no encryption key.");

      // Encrypted in the browser; only this ciphertext ever goes on chain.
      const ciphertext = encryptTriggerPrice(pubKey, trigger6.toString());

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
      const message =
        err instanceof Error ? err.message.split("\n")[0] : String(err);
      setPhase({ kind: "error", message });
    }
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-4">
      <label className="flex flex-col gap-1.5">
        <span className="text-sm text-zinc-400">
          Deposit to protect ({chain.nativeCurrency.symbol})
        </span>
        <input
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          required
          inputMode="decimal"
          pattern="[0-9]*\.?[0-9]+"
          className="rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 font-mono text-zinc-100 outline-none transition focus:border-emerald-500"
        />
      </label>

      <label className="flex flex-col gap-1.5">
        <span className="text-sm text-zinc-400">
          Trigger price (USD per FLR) — sell if price drops to or below
        </span>
        <input
          value={trigger}
          onChange={(e) => setTrigger(e.target.value)}
          required
          inputMode="decimal"
          pattern="[0-9]*\.?[0-9]+"
          className="rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 font-mono text-zinc-100 outline-none transition focus:border-emerald-500"
        />
      </label>

      <p className="text-xs text-zinc-500">
        Trigger price is ECIES-encrypted in your browser to the TEE&apos;s
        enclave key before it is sent.
        {fee !== undefined && (
          <> Instruction fee: {fee.toString()} wei (added to your deposit).</>
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
            : "Place encrypted stop-loss"}
      </button>

      {!isConnected && (
        <p className="text-xs text-zinc-500">Connect a wallet to place orders.</p>
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
