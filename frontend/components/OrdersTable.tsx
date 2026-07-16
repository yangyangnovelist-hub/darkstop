"use client";

import { useState } from "react";
import { formatUnits } from "viem";
import { useAccount, useReadContract, useWriteContract } from "wagmi";
import { txUrl } from "@/lib/chain";
import { VAULT_ADDRESS, vaultAbi } from "@/lib/vault";
import { useOrders, type OrderRow } from "./useOrders";

const STATUS_STYLE: Record<OrderRow["status"], string> = {
  Pending: "bg-amber-500/15 text-amber-400 border-amber-500/30",
  Executed: "bg-emerald-500/15 text-emerald-400 border-emerald-500/30",
  Cancelled: "bg-zinc-500/15 text-zinc-400 border-zinc-500/30",
};

function TxLink({ hash, label }: { hash?: string; label: string }) {
  if (!hash) return null;
  const url = txUrl(hash);
  if (!url) {
    return (
      <span className="font-mono text-xs text-zinc-500" title={hash}>
        {label} {hash.slice(0, 10)}…
      </span>
    );
  }
  return (
    <a
      href={url}
      target="_blank"
      rel="noreferrer"
      className="text-xs text-zinc-400 underline decoration-zinc-600 underline-offset-2 hover:text-zinc-200"
    >
      {label} ↗
    </a>
  );
}

export function OrdersTable() {
  const { orders, historyError } = useOrders();
  const { address } = useAccount();
  const { data: fee } = useReadContract({
    address: VAULT_ADDRESS,
    abi: vaultAbi,
    functionName: "INSTRUCTION_FEE",
  });
  const { writeContractAsync } = useWriteContract();
  const [cancelling, setCancelling] = useState<string | null>(null);
  const [cancelError, setCancelError] = useState<string | null>(null);

  async function cancelOrder(id: bigint) {
    if (fee === undefined) return;
    setCancelError(null);
    setCancelling(id.toString());
    try {
      await writeContractAsync({
        address: VAULT_ADDRESS,
        abi: vaultAbi,
        functionName: "cancel",
        args: [id],
        value: fee, // forwarded to the registry with the CANCEL_ORDER instruction
      });
    } catch (err) {
      const message =
        err instanceof Error ? err.message.split("\n")[0] : String(err);
      setCancelError(message);
    } finally {
      setCancelling(null);
    }
  }

  if (orders.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-zinc-500">
        No orders yet. Events stream in live once the first order is placed.
        {historyError && (
          <span className="mt-2 block text-xs text-amber-500">{historyError}</span>
        )}
      </p>
    );
  }

  return (
    <div className="overflow-x-auto">
      {historyError && (
        <p className="mb-2 text-xs text-amber-500">{historyError}</p>
      )}
      {cancelError && (
        <p className="mb-2 break-words text-xs text-red-400">{cancelError}</p>
      )}
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-zinc-800 text-xs uppercase tracking-wider text-zinc-500">
            <th className="py-2 pr-4">Order</th>
            <th className="py-2 pr-4">Owner</th>
            <th className="py-2 pr-4">Status</th>
            <th className="py-2 pr-4">Settled price</th>
            <th className="py-2 pr-4">Txs</th>
            <th className="py-2" />
          </tr>
        </thead>
        <tbody>
          {orders.map((o) => (
            <tr key={o.id.toString()} className="border-b border-zinc-900">
              <td className="py-3 pr-4 font-mono">#{o.id.toString()}</td>
              <td className="py-3 pr-4 font-mono text-zinc-400">
                {o.owner ? `${o.owner.slice(0, 6)}…${o.owner.slice(-4)}` : "—"}
              </td>
              <td className="py-3 pr-4">
                <span
                  className={`inline-block rounded-full border px-2.5 py-0.5 text-xs font-medium ${STATUS_STYLE[o.status]}`}
                >
                  {o.status}
                </span>
              </td>
              <td className="py-3 pr-4 font-mono text-zinc-300">
                {o.price !== undefined ? `$${formatUnits(o.price, 6)}` : "—"}
              </td>
              <td className="py-3 pr-4">
                <div className="flex gap-3">
                  <TxLink hash={o.placedTx} label="placed" />
                  <TxLink hash={o.finalTx} label={o.status.toLowerCase()} />
                </div>
              </td>
              <td className="py-3 text-right">
                {o.status === "Pending" &&
                  address &&
                  o.owner?.toLowerCase() === address.toLowerCase() && (
                    <button
                      onClick={() => cancelOrder(o.id)}
                      disabled={cancelling === o.id.toString()}
                      className="rounded-md border border-zinc-700 px-2.5 py-1 text-xs text-zinc-300 transition hover:border-red-500/60 hover:text-red-400 disabled:opacity-50"
                    >
                      {cancelling === o.id.toString() ? "Cancelling…" : "Cancel"}
                    </button>
                  )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
