import { createConfig, http } from "wagmi";
import { injected } from "wagmi/connectors/injected";
import { chain } from "./chain";

export const wagmiConfig = createConfig({
  chains: [chain],
  connectors: [injected()],
  transports: { [chain.id]: http() },
});
