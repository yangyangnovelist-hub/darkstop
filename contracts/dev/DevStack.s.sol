// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import { Script } from "forge-std/Script.sol";
import { console2 } from "forge-std/console2.sol";
import { DarkStopVault } from "../DarkStopVault.sol";
import { MockUSDT0 } from "../MockUSDT0.sol";
import { ITeeExtensionRegistry } from "../interfaces/ITeeExtensionRegistry.sol";
import { ITeeMachineRegistry } from "../interfaces/ITeeMachineRegistry.sol";
import { MockTeeExtensionRegistry, MockTeeMachineRegistry, MockFtsoV2 } from "./Mocks.sol";

/// @notice Deploys the full DarkStop stack against mock TEE/FTSO dependencies
/// on a local anvil chain. Driven by scripts/dev-stack.sh, which parses the
/// ADDR lines below into frontend/.env.local. Local development only.
contract DevStack is Script {
    uint256 internal constant INSTRUCTION_FEE = 0.01 ether;

    function run() external {
        vm.startBroadcast();

        MockTeeExtensionRegistry extensionRegistry = new MockTeeExtensionRegistry();
        MockTeeMachineRegistry machineRegistry = new MockTeeMachineRegistry();
        MockFtsoV2 ftso = new MockFtsoV2();
        MockUSDT0 usdt0 = new MockUSDT0();

        DarkStopVault vault = new DarkStopVault(
            ITeeExtensionRegistry(address(extensionRegistry)),
            ITeeMachineRegistry(address(machineRegistry)),
            address(ftso),
            address(usdt0),
            INSTRUCTION_FEE
        );

        extensionRegistry.setRegisteredSender(address(vault));
        vault.setExtensionId();
        // Testnet convention (mirrors Coston2): executor = deployer, so
        // settle() can be simulated with a plain `cast send`.
        vault.setTeeExecutor(msg.sender);

        // 1,000,000 USDT0 payout pool + a fresh FLR/USD price of $0.02
        // (value 200000, feed decimals 7 — same fixture as the unit tests).
        usdt0.mint(address(vault), 1_000_000e6);
        ftso.setFeed(200_000, 7, uint64(block.timestamp));

        vm.stopBroadcast();

        console2.log("ADDR_VAULT=%s", address(vault));
        console2.log("ADDR_USDT0=%s", address(usdt0));
        console2.log("ADDR_FTSO=%s", address(ftso));
        console2.log("ADDR_EXT_REGISTRY=%s", address(extensionRegistry));
        console2.log("ADDR_MACHINE_REGISTRY=%s", address(machineRegistry));
    }
}
