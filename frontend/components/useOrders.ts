"use client";

import { useCallback, useEffect, useState } from "react";
import { usePublicClient } from "wagmi";
import { VAULT_ADDRESS, vaultAbi } from "@/lib/vault";

export type OrderStatus = "Pending" | "Executed" | "Cancelled";

export type OrderRow = {
  id: bigint;
  owner?: `0x${string}`;
  status: OrderStatus;
  /** Settlement price (USD/FLR, 6 decimals), set on execution. */
  price?: bigint;
  placedTx?: string;
  /** Tx that executed or cancelled the order. */
  finalTx?: string;
  /** Block containing the execution or cancellation receipt. */
  finalBlock?: bigint;
};

type VaultLog = {
  eventName: "OrderPlaced" | "OrderExecuted" | "OrderCancelled";
  args: { orderId?: bigint; owner?: `0x${string}`; price?: bigint };
  transactionHash: `0x${string}` | null;
  blockNumber: bigint | null;
};

/** Live order book derived purely from vault events: history via getLogs,
 *  then a viem watchContractEvent subscription keeps it current. */
export function useOrders() {
  const client = usePublicClient();
  const [orders, setOrders] = useState<Map<string, OrderRow>>(new Map());
  const [historyError, setHistoryError] = useState<string | null>(null);

  const apply = useCallback((logs: VaultLog[]) => {
    setOrders((prev) => {
      const next = new Map(prev);
      for (const log of logs) {
        const id = log.args.orderId;
        if (id === undefined) continue;
        const key = id.toString();
        const row: OrderRow = next.get(key) ?? { id, status: "Pending" };
        switch (log.eventName) {
          case "OrderPlaced":
            row.owner = log.args.owner;
            row.placedTx = log.transactionHash ?? row.placedTx;
            break;
          case "OrderExecuted":
            row.status = "Executed";
            row.price = log.args.price;
            row.finalTx = log.transactionHash ?? row.finalTx;
            row.finalBlock = log.blockNumber ?? row.finalBlock;
            break;
          case "OrderCancelled":
            row.status = "Cancelled";
            row.finalTx = log.transactionHash ?? row.finalTx;
            row.finalBlock = log.blockNumber ?? row.finalBlock;
            break;
        }
        next.set(key, row);
      }
      return next;
    });
  }, []);

  useEffect(() => {
    if (!client) return;
    let cancelled = false;

    (async () => {
      // Full history from genesis first; if the RPC caps getLogs ranges,
      // fall back to a recent window (live watcher still covers new events).
      const latest = await client.getBlockNumber().catch(() => 0n);
      const envStart = process.env.NEXT_PUBLIC_START_BLOCK;
      const starts = envStart
        ? [BigInt(envStart)]
        : [0n, latest > 20_000n ? latest - 20_000n : 0n];
      for (const fromBlock of starts) {
        try {
          const logs = await client.getContractEvents({
            address: VAULT_ADDRESS,
            abi: vaultAbi,
            fromBlock,
            toBlock: "latest",
          });
          if (!cancelled) {
            apply(logs as unknown as VaultLog[]);
            setHistoryError(null);
          }
          return;
        } catch {
          // try the next, narrower window
        }
      }
      if (!cancelled) {
        setHistoryError("Could not load order history; showing live events only.");
      }
    })();

    const unwatch = client.watchContractEvent({
      address: VAULT_ADDRESS,
      abi: vaultAbi,
      pollingInterval: 2_000,
      onLogs: (logs) => apply(logs as unknown as VaultLog[]),
    });

    return () => {
      cancelled = true;
      unwatch();
    };
  }, [client, apply]);

  const rows = [...orders.values()].sort((a, b) => (a.id < b.id ? 1 : -1));
  return { orders: rows, historyError };
}
