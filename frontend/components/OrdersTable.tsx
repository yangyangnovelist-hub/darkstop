"use client";

import { Fragment, useState } from "react";
import { formatUnits } from "viem";
import { useAccount, useReadContract, useWriteContract } from "wagmi";
import { chain, txUrl } from "@/lib/chain";
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
    <div className="min-w-0">
      {historyError && (
        <p className="mb-2 text-xs text-amber-500">{historyError}</p>
      )}
      {cancelError && (
        <p className="mb-2 break-words text-xs text-red-400">{cancelError}</p>
      )}

      <div className="grid gap-3 sm:hidden">
        {orders.map((o) => (
          <article
            key={o.id.toString()}
            aria-label={`Order ${o.id.toString()}`}
            className="min-w-0 rounded-2xl border border-zinc-800 bg-black/20 p-4"
          >
            <div className="flex items-center justify-between gap-3">
              <span className="font-mono text-sm text-zinc-200">
                #{o.id.toString()}
              </span>
              <span
                className={`inline-block rounded-full border px-2.5 py-0.5 text-xs font-medium ${STATUS_STYLE[o.status]}`}
              >
                {o.status}
              </span>
            </div>

            <dl className="mt-4 grid grid-cols-2 gap-3 text-xs">
              <div className="min-w-0">
                <dt className="uppercase tracking-wider text-zinc-600">Owner</dt>
                <dd className="mt-1 truncate font-mono text-zinc-300" title={o.owner}>
                  {o.owner ? `${o.owner.slice(0, 6)}…${o.owner.slice(-4)}` : "—"}
                </dd>
              </div>
              <div>
                <dt className="uppercase tracking-wider text-zinc-600">Settled price</dt>
                <dd className="mt-1 font-mono text-zinc-200">
                  {o.price !== undefined ? `$${formatUnits(o.price, 6)}` : "—"}
                </dd>
              </div>
            </dl>

            <div className="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-zinc-900 pt-3">
              <div className="flex flex-wrap gap-3">
                <TxLink hash={o.placedTx} label="placed" />
                <TxLink hash={o.finalTx} label={o.status.toLowerCase()} />
              </div>
              {o.status === "Pending" &&
                address &&
                o.owner?.toLowerCase() === address.toLowerCase() && (
                  <button
                    onClick={() => cancelOrder(o.id)}
                    disabled={cancelling === o.id.toString()}
                    className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300 transition hover:border-red-500/60 hover:text-red-400 disabled:opacity-50"
                  >
                    {cancelling === o.id.toString() ? "Cancelling…" : "Cancel"}
                  </button>
                )}
            </div>

            {o.status === "Executed" && (
              <div className="mt-3">
                <ExecutionProofs order={o} />
              </div>
            )}
          </article>
        ))}
      </div>

      <div className="hidden min-w-0 overflow-x-auto sm:block">
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
              <Fragment key={o.id.toString()}>
                <tr className={o.status === "Executed" ? "border-0" : "border-b border-zinc-900"}>
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
                {o.status === "Executed" && (
                  <tr className="border-b border-zinc-900">
                    <td colSpan={6} className="pb-4 pt-1">
                      <ExecutionProofs order={o} />
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function ExecutionProofs({ order }: { order: OrderRow }) {
  const { data: executor } = useReadContract({
    address: VAULT_ADDRESS,
    abi: vaultAbi,
    functionName: "teeExecutor",
  });
  const { data: chainOrder } = useReadContract({
    address: VAULT_ADDRESS,
    abi: vaultAbi,
    functionName: "orders",
    args: [order.id],
  });
  const deposit = chainOrder?.[1];
  const payout = deposit !== undefined && order.price !== undefined
    ? deposit * order.price / 10n ** 18n
    : undefined;

  return (
    <div className="grid gap-2 rounded-xl border border-emerald-500/15 bg-emerald-500/[0.035] p-3 sm:grid-cols-2 xl:grid-cols-5">
      <Proof label="Network" value={chain.name} />
      <Proof label="Allowed executor" value={executor ? `${executor.slice(0, 6)}…${executor.slice(-4)}` : "loading…"} />
      <Proof label="FTSO settlement" value={order.price !== undefined ? `$${formatUnits(order.price, 6)}` : "—"} />
      <Proof label="Test payout" value={payout !== undefined ? `${formatUnits(payout, 6)} USDT0` : "loading…"} />
      <Proof label="Receipt" value={order.finalBlock !== undefined ? `block #${order.finalBlock}` : order.finalTx ? `${order.finalTx.slice(0, 10)}…` : "—"} />
    </div>
  );
}

function Proof({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-2 text-xs">
      <span className="grid size-5 place-items-center rounded-full bg-emerald-400 text-[10px] font-black text-black">✓</span>
      <span><span className="block text-zinc-500">{label}</span><span className="font-medium text-emerald-200">{value}</span></span>
    </div>
  );
}
