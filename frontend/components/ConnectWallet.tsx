"use client";

import { useAccount, useConnect, useDisconnect, useSwitchChain } from "wagmi";
import { chain } from "@/lib/chain";

export function ConnectWallet() {
  const { address, isConnected, chainId } = useAccount();
  const { connect, connectors, isPending, error } = useConnect();
  const { disconnect } = useDisconnect();
  const { switchChain, isPending: isSwitching } = useSwitchChain();

  if (!isConnected) {
    const walletError = connectors.length === 0
      ? "No browser wallet detected. Install an injected wallet to continue."
      : error
        ? "Wallet connection failed. Open your wallet and try again."
        : null;
    return (
      <div className="relative flex flex-col items-end">
        <button
          onClick={() => connect({ connector: connectors[0] })}
          disabled={isPending || connectors.length === 0}
          className="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-black transition hover:bg-emerald-400 disabled:opacity-50"
        >
          {isPending ? "Connecting…" : "Connect wallet"}
        </button>
        {walletError && (
          <p role="alert" className="absolute right-0 top-full z-20 mt-2 w-64 rounded-lg border border-red-500/20 bg-zinc-950 p-3 text-xs text-red-300 shadow-xl">
            {walletError}
          </p>
        )}
      </div>
    );
  }

  if (chainId !== chain.id) {
    return (
      <button
        onClick={() => switchChain({ chainId: chain.id })}
        disabled={isSwitching}
        className="rounded-lg bg-amber-500 px-4 py-2 text-sm font-semibold text-black transition hover:bg-amber-400 disabled:opacity-50"
      >
        {isSwitching ? "Switching…" : `Switch to ${chain.name}`}
      </button>
    );
  }

  return (
    <div className="flex items-center gap-3">
      <span className="font-mono text-sm text-zinc-400">
        {address?.slice(0, 6)}…{address?.slice(-4)}
      </span>
      <button
        onClick={() => disconnect()}
        className="rounded-lg border border-zinc-700 px-3 py-2 text-sm text-zinc-300 transition hover:border-zinc-500"
      >
        Disconnect
      </button>
    </div>
  );
}
