// Minimal ABI surface of contracts/DarkStopVault.sol used by the UI.
export const VAULT_ADDRESS = (process.env.NEXT_PUBLIC_VAULT_ADDRESS ??
  "0xd93E8F7dE2A5A7C4eC45F115f7047103da2dD8bF") as `0x${string}`; // Coston2 (docs/deployments.md)

export const vaultAbi = [
  {
    type: "function",
    name: "placeOrder",
    stateMutability: "payable",
    inputs: [{ name: "_ciphertext", type: "bytes" }],
    outputs: [{ name: "id", type: "uint256" }],
  },
  {
    type: "function",
    name: "cancel",
    stateMutability: "payable",
    inputs: [{ name: "_orderId", type: "uint256" }],
    outputs: [],
  },
  {
    type: "function",
    name: "INSTRUCTION_FEE",
    stateMutability: "view",
    inputs: [],
    outputs: [{ type: "uint256" }],
  },
  {
    type: "event",
    name: "OrderPlaced",
    inputs: [
      { name: "orderId", type: "uint256", indexed: true },
      { name: "owner", type: "address", indexed: true },
    ],
  },
  {
    type: "event",
    name: "OrderExecuted",
    inputs: [
      { name: "orderId", type: "uint256", indexed: true },
      { name: "price", type: "uint256", indexed: false },
    ],
  },
  {
    type: "event",
    name: "OrderCancelled",
    inputs: [{ name: "orderId", type: "uint256", indexed: true }],
  },
] as const;
