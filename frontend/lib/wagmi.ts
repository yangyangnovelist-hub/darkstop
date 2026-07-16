import { createConfig, http } from "wagmi";
import { injected } from "wagmi/connectors";
import { chain } from "./chain";

export const wagmiConfig = createConfig({
  chains: [chain],
  connectors: [injected()],
  transports: { [chain.id]: http() },
});
