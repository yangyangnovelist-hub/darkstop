import { defineChain } from "viem";

// Defaults target Flare Testnet Coston2; scripts/dev-stack.sh overrides these
// via frontend/.env.local to point at a local anvil devnet.
const chainId = Number(process.env.NEXT_PUBLIC_CHAIN_ID ?? "114");
const rpcUrl =
  process.env.NEXT_PUBLIC_RPC_URL ??
  "https://coston2-api.flare.network/ext/C/rpc";

/** Block-explorer base URL; empty string on local devnets (no explorer). */
export const explorerBase =
  process.env.NEXT_PUBLIC_EXPLORER_URL ??
  (chainId === 114 ? "https://coston2-explorer.flare.network" : "");

export const chain = defineChain({
  id: chainId,
  name: chainId === 114 ? "Flare Testnet Coston2" : `Local devnet (${chainId})`,
  nativeCurrency: {
    name: chainId === 114 ? "Coston2 Flare" : "Devnet Ether",
    symbol: chainId === 114 ? "C2FLR" : "ETH",
    decimals: 18,
  },
  rpcUrls: { default: { http: [rpcUrl] } },
  ...(explorerBase
    ? { blockExplorers: { default: { name: "Explorer", url: explorerBase } } }
    : {}),
  testnet: true,
});

export function txUrl(hash: string): string | null {
  return explorerBase ? `${explorerBase}/tx/${hash}` : null;
}

export function addressUrl(address: string): string | null {
  return explorerBase ? `${explorerBase}/address/${address}` : null;
}
