// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import { Test } from "forge-std/Test.sol";
import { DarkStopVault } from "../contracts/DarkStopVault.sol";
import { MockUSDT0 } from "../contracts/MockUSDT0.sol";
import { ITeeExtensionRegistry } from "../contracts/interfaces/ITeeExtensionRegistry.sol";
import { ITeeMachineRegistry } from "../contracts/interfaces/ITeeMachineRegistry.sol";
import { MockTeeExtensionRegistry, MockTeeMachineRegistry } from "./DarkStopVault.t.sol";
import { ContractRegistry } from "flare-periphery/coston2/ContractRegistry.sol";
import { TestFtsoV2Interface } from "flare-periphery/coston2/TestFtsoV2Interface.sol";

/// @notice Coston2 fork test: exercises the vault against the REAL FtsoV2
/// (resolved exactly like production: ContractRegistry.getTestFtsoV2()).
/// TEE registries stay mocked — machine registration on Coston2 is gated on
/// the extension proxy, and these tests verify the FTSO/settlement path only.
///
/// Run with:
///   forge test --match-contract DarkStopVaultForkTest --fork-url "$CHAIN_URL"
///
/// Without a Coston2 fork (chain id 114) every test self-skips.
contract DarkStopVaultForkTest is Test {
    event OrderPlaced(uint256 indexed orderId, address indexed owner);
    event OrderExecuted(uint256 indexed orderId, uint256 price);

    uint256 internal constant INSTRUCTION_FEE = 1000000 wei;

    bool internal forked;

    MockTeeExtensionRegistry internal extensionRegistry;
    MockTeeMachineRegistry internal machineRegistry;
    TestFtsoV2Interface internal ftso;
    MockUSDT0 internal usdt0;
    DarkStopVault internal vault;

    address internal alice = makeAddr("alice");
    address internal executor = makeAddr("executor");

    bytes internal ciphertext = hex"deadbeef0102030405";

    function setUp() public {
        forked = block.chainid == 114; // Coston2
        if (!forked) return;

        // Production code path: resolve FtsoV2 from the on-chain registry.
        ftso = ContractRegistry.getTestFtsoV2();

        extensionRegistry = new MockTeeExtensionRegistry();
        machineRegistry = new MockTeeMachineRegistry();
        usdt0 = new MockUSDT0();

        vault = new DarkStopVault(
            ITeeExtensionRegistry(address(extensionRegistry)),
            ITeeMachineRegistry(address(machineRegistry)),
            address(ftso),
            address(usdt0),
            INSTRUCTION_FEE
        );
        extensionRegistry.setRegisteredSender(address(vault));
        vault.setExtensionId();
        vault.setTeeExecutor(executor);

        usdt0.mint(address(vault), 1_000_000e6);
        vm.deal(alice, 10 ether);
    }

    /// @dev Live FLR/USD price normalized to the vault's 6 payout decimals.
    function livePrice() internal view returns (uint256 price, uint64 ts) {
        (uint256 value, int8 decimals, uint64 timestamp) = ftso.getFeedById(vault.FLR_USD());
        int256 shift = int256(6) - int256(decimals);
        price = shift >= 0 ? value * 10 ** uint256(shift) : value / 10 ** uint256(-shift);
        ts = timestamp;
    }

    function test_Fork_FeedIsLiveAndFresh() public {
        vm.skip(!forked);
        (uint256 price, uint64 ts) = livePrice();
        assertGt(price, 0, "live FLR/USD price should be positive");
        assertLe(block.timestamp - ts, 300, "live feed should be fresh at the fork block");
    }

    function test_Fork_PlaceAndSettleAgainstLiveFtso() public {
        vm.skip(!forked);

        vm.prank(alice);
        uint256 id = vault.placeOrder{value: 0.5 ether + INSTRUCTION_FEE}(ciphertext);
        (, uint256 deposit, uint8 status) = vault.orders(id);
        assertEq(status, vault.STATUS_OPEN());
        assertEq(deposit, 0.5 ether);

        // Trigger just above the live price -> settle must pass the FTSO re-check.
        (uint256 price,) = livePrice();
        vm.prank(executor);
        vault.settle(id, price + 1, 300);

        (,, status) = vault.orders(id);
        assertEq(status, vault.STATUS_EXECUTED());
        // payout = 0.5e18 * price / 1e18 (6-decimals USDT0)
        assertEq(usdt0.balanceOf(alice), price / 2);
    }

    function test_Fork_SettleRevertsWhenPriceAboveTrigger() public {
        vm.skip(!forked);

        vm.prank(alice);
        uint256 id = vault.placeOrder{value: 0.5 ether + INSTRUCTION_FEE}(ciphertext);

        (uint256 price,) = livePrice();
        vm.prank(executor);
        vm.expectRevert(bytes("price above trigger"));
        vault.settle(id, price / 2, 300);
    }

    function test_Fork_SettleRevertsOnStaleFeed() public {
        vm.skip(!forked);

        vm.prank(alice);
        uint256 id = vault.placeOrder{value: 0.5 ether + INSTRUCTION_FEE}(ciphertext);

        (uint256 price,) = livePrice();
        // The fork pins the feed timestamp; an hour later it is stale.
        vm.warp(block.timestamp + 3600);
        vm.prank(executor);
        vm.expectRevert(bytes("stale price"));
        vault.settle(id, price + 1, 300);
    }
}
