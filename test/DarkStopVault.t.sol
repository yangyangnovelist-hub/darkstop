// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import { Test } from "forge-std/Test.sol";
import { DarkStopVault } from "../contracts/DarkStopVault.sol";
import { MockUSDT0 } from "../contracts/MockUSDT0.sol";
import { ITeeExtensionRegistry } from "../contracts/interfaces/ITeeExtensionRegistry.sol";
import { ITeeMachineRegistry } from "../contracts/interfaces/ITeeMachineRegistry.sol";
// Mocks live in contracts/dev/ so the local dev stack (contracts/dev/DevStack.s.sol)
// deploys byte-identical dependencies to what this suite tests against.
import { MockTeeExtensionRegistry, MockTeeMachineRegistry, MockFtsoV2 } from "../contracts/dev/Mocks.sol";

contract DarkStopVaultTest is Test {
    event OrderPlaced(uint256 indexed orderId, address indexed owner);
    event OrderExecuted(uint256 indexed orderId, uint256 price);
    event OrderCancelled(uint256 indexed orderId);

    uint256 internal constant INSTRUCTION_FEE = 0.01 ether;

    MockTeeExtensionRegistry internal extensionRegistry;
    MockTeeMachineRegistry internal machineRegistry;
    MockFtsoV2 internal ftso;
    MockUSDT0 internal usdt0;
    DarkStopVault internal vault;

    address internal alice = makeAddr("alice");
    address internal bob = makeAddr("bob");
    address internal executor = makeAddr("executor");

    bytes internal ciphertext = hex"deadbeef0102030405";

    function setUp() public {
        extensionRegistry = new MockTeeExtensionRegistry();
        machineRegistry = new MockTeeMachineRegistry();
        ftso = new MockFtsoV2();
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

        // Deterministic base time for staleness checks.
        vm.warp(1_000_000);

        // Pre-fund the vault's payout pool with 1,000,000 USDT0 (6 decimals).
        usdt0.mint(address(vault), 1_000_000e6);

        vm.deal(alice, 100 ether);
        vm.deal(bob, 100 ether);
    }

    // ---------------------------------------------------------------
    // Constants (must match Go extension config byte-for-byte)
    // ---------------------------------------------------------------

    function test_Constants() public view {
        assertEq(vault.OP_TYPE_DARKSTOP(), bytes32("DARKSTOP"));
        assertEq(vault.OP_COMMAND_PLACE(), bytes32("PLACE_ORDER"));
        assertEq(vault.OP_COMMAND_CANCEL(), bytes32("CANCEL_ORDER"));
        assertEq(vault.FLR_USD(), bytes21(0x01464c522f55534400000000000000000000000000));
        assertEq(vault.MAX_PRICE_AGE_SEC(), 300);
    }

    // ---------------------------------------------------------------
    // placeOrder
    // ---------------------------------------------------------------

    function test_PlaceOrder_StoresDepositAndEmitsEvent() public {
        vm.prank(alice);
        vm.expectEmit(true, true, true, true, address(vault));
        emit OrderPlaced(1, alice);
        uint256 id = vault.placeOrder{value: INSTRUCTION_FEE + 5 ether}(ciphertext);

        assertEq(id, 1);
        (address owner_, uint256 deposit, uint8 status) = vault.orders(id);
        assertEq(owner_, alice);
        assertEq(deposit, 5 ether);
        assertEq(status, 1); // open
        assertEq(address(vault).balance, 5 ether); // deposit held, fee forwarded
    }

    function test_PlaceOrder_SendsInstructionWithFee() public {
        vm.prank(alice);
        uint256 id = vault.placeOrder{value: INSTRUCTION_FEE + 5 ether}(ciphertext);

        assertEq(extensionRegistry.sendCount(), 1);
        assertEq(extensionRegistry.lastOpType(), bytes32("DARKSTOP"));
        assertEq(extensionRegistry.lastOpCommand(), bytes32("PLACE_ORDER"));
        assertEq(extensionRegistry.lastValue(), INSTRUCTION_FEE);
        assertEq(extensionRegistry.lastTeeIdCount(), 1);
        assertEq(extensionRegistry.lastClaimBackAddress(), alice);

        // Message is abi.encode(orderId, ciphertext) — the ciphertext is the
        // ONLY order-specific payload; no price data anywhere on chain.
        assertEq(extensionRegistry.lastMessage(), abi.encode(id, ciphertext));
    }

    function test_PlaceOrder_NoPriceDataOnChain() public {
        vm.prank(alice);
        uint256 id = vault.placeOrder{value: INSTRUCTION_FEE + 5 ether}(ciphertext);

        // The order struct stores only owner / deposit / status.
        (address owner_, uint256 deposit, uint8 status) = vault.orders(id);
        assertEq(owner_, alice);
        assertEq(deposit, 5 ether);
        assertEq(status, 1);

        // The instruction message is exactly (id, opaque ciphertext): decoding
        // it yields back the ciphertext bytes unchanged, i.e. the contract
        // never parses or stores anything derived from the encrypted payload.
        (uint256 decodedId, bytes memory decodedCiphertext) =
            abi.decode(extensionRegistry.lastMessage(), (uint256, bytes));
        assertEq(decodedId, id);
        assertEq(decodedCiphertext, ciphertext);
    }

    function test_PlaceOrder_IdsIncrement() public {
        vm.prank(alice);
        uint256 first = vault.placeOrder{value: INSTRUCTION_FEE + 1 ether}(ciphertext);
        vm.prank(bob);
        uint256 second = vault.placeOrder{value: INSTRUCTION_FEE + 2 ether}(ciphertext);

        assertEq(first, 1);
        assertEq(second, 2);
        assertEq(vault.nextOrderId(), 2);

        (address secondOwner, uint256 secondDeposit,) = vault.orders(second);
        assertEq(secondOwner, bob);
        assertEq(secondDeposit, 2 ether);
    }

    function test_PlaceOrder_RevertsIfValueNotAboveFee() public {
        vm.prank(alice);
        vm.expectRevert(bytes("value must exceed instruction fee"));
        vault.placeOrder{value: INSTRUCTION_FEE}(ciphertext);
    }

    function test_PlaceOrder_RevertsOnEmptyCiphertext() public {
        vm.prank(alice);
        vm.expectRevert(bytes("empty ciphertext"));
        vault.placeOrder{value: INSTRUCTION_FEE + 1 ether}("");
    }

    // ---------------------------------------------------------------
    // setTeeExecutor
    // ---------------------------------------------------------------

    function test_SetTeeExecutor_OnlyOwner() public {
        vm.prank(bob);
        vm.expectRevert(bytes("not contract owner"));
        vault.setTeeExecutor(bob);

        // Contract owner (this test contract deployed the vault) may set it.
        vault.setTeeExecutor(address(0x1234));
        assertEq(vault.teeExecutor(), address(0x1234));
    }

    // ---------------------------------------------------------------
    // settle
    //
    // Price conventions: triggerPrice is USD per FLR scaled to the payout
    // token's 6 decimals. The FTSO feed value comes with its own int8
    // decimals and is normalized to 6 decimals before comparison/payout.
    // ---------------------------------------------------------------

    /// @dev Places a 5-ether-deposit order for alice and sets a fresh FTSO
    /// price of 0.02 USD/FLR published with feed decimals 7 (value 200000).
    function _placeAliceOrder() internal returns (uint256 id) {
        vm.prank(alice);
        id = vault.placeOrder{value: INSTRUCTION_FEE + 5 ether}(ciphertext);
        ftso.setFeed(200_000, 7, uint64(block.timestamp));
    }

    function test_Settle_OnlyTeeExecutor() public {
        uint256 id = _placeAliceOrder();

        vm.prank(bob);
        vm.expectRevert(bytes("not tee executor"));
        vault.settle(id, 25_000, 300);

        vm.prank(alice); // even the order owner cannot settle
        vm.expectRevert(bytes("not tee executor"));
        vault.settle(id, 25_000, 300);
    }

    function test_Settle_RevertsOnStalePrice() public {
        uint256 id = _placeAliceOrder();
        vm.warp(block.timestamp + 301); // feed timestamp now 301s old, max 300

        vm.prank(executor);
        vm.expectRevert(bytes("stale price"));
        vault.settle(id, 25_000, 300);
    }

    function test_Settle_RevertsIfExecutorTriesToRelaxPriceFreshness() public {
        uint256 id = _placeAliceOrder();

        vm.prank(executor);
        vm.expectRevert(bytes("max age exceeds protocol limit"));
        vault.settle(id, 25_000, 301);
    }

    function test_Settle_RevertsIfPriceAboveTrigger() public {
        uint256 id = _placeAliceOrder(); // current price 0.02 = 20_000 (6 dec)

        vm.prank(executor);
        vm.expectRevert(bytes("price above trigger"));
        vault.settle(id, 19_999, 300); // trigger just below current price
    }

    function test_Settle_PaysOutAndFlipsStatus() public {
        uint256 id = _placeAliceOrder();

        // deposit 5e18 wei * price 20_000 (0.02 USD, 6 dec) / 1e18
        // = 100_000 = 0.1 USDT0
        vm.prank(executor);
        vm.expectEmit(true, true, true, true, address(vault));
        emit OrderExecuted(id, 20_000);
        vault.settle(id, 25_000, 300);

        assertEq(usdt0.balanceOf(alice), 100_000);
        (, uint256 deposit, uint8 status) = vault.orders(id);
        assertEq(status, 2); // executed
        assertEq(deposit, 5 ether); // deposit stays in the vault
        assertEq(address(vault).balance, 5 ether);
    }

    function test_Settle_NormalizesLowDecimalFeeds() public {
        vm.prank(alice);
        uint256 id = vault.placeOrder{value: INSTRUCTION_FEE + 5 ether}(ciphertext);

        // Same 0.02 USD price but published with 5 feed decimals, and exactly
        // maxAgeSec old (boundary: still fresh).
        ftso.setFeed(2_000, 5, uint64(block.timestamp));
        vm.warp(block.timestamp + 300);

        vm.prank(executor);
        vault.settle(id, 20_000, 300); // trigger == price: at-or-below triggers

        assertEq(usdt0.balanceOf(alice), 100_000);
    }

    function test_Settle_TwiceReverts() public {
        uint256 id = _placeAliceOrder();

        vm.prank(executor);
        vault.settle(id, 25_000, 300);

        vm.prank(executor);
        vm.expectRevert(bytes("order not open"));
        vault.settle(id, 25_000, 300);
    }

    function test_Settle_RevertsOnUnknownOrder() public {
        ftso.setFeed(200_000, 7, uint64(block.timestamp));
        vm.prank(executor);
        vm.expectRevert(bytes("order not open"));
        vault.settle(42, 25_000, 300);
    }

    // ---------------------------------------------------------------
    // cancel
    // ---------------------------------------------------------------

    function test_Cancel_OnlyOrderOwner() public {
        uint256 id = _placeAliceOrder();

        vm.prank(bob);
        vm.expectRevert(bytes("not order owner"));
        vault.cancel(id);
    }

    function test_Cancel_RefundsDepositAndSendsInstruction() public {
        uint256 id = _placeAliceOrder();
        uint256 balanceBefore = alice.balance;

        vm.prank(alice);
        vm.expectEmit(true, true, true, true, address(vault));
        emit OrderCancelled(id);
        vault.cancel(id);

        assertEq(alice.balance, balanceBefore + 5 ether); // full deposit back
        assertEq(address(vault).balance, 0);
        (,, uint8 status) = vault.orders(id);
        assertEq(status, 3); // cancelled

        // CANCEL_ORDER instruction sent (2nd instruction after PLACE_ORDER).
        assertEq(extensionRegistry.sendCount(), 2);
        assertEq(extensionRegistry.lastOpType(), bytes32("DARKSTOP"));
        assertEq(extensionRegistry.lastOpCommand(), bytes32("CANCEL_ORDER"));
        assertEq(extensionRegistry.lastMessage(), abi.encode(id));
    }

    function test_Cancel_ThenSettleReverts() public {
        uint256 id = _placeAliceOrder();

        vm.prank(alice);
        vault.cancel(id);

        vm.prank(executor);
        vm.expectRevert(bytes("order not open"));
        vault.settle(id, 25_000, 300);
    }

    function test_Cancel_AfterExecutedReverts() public {
        uint256 id = _placeAliceOrder();

        vm.prank(executor);
        vault.settle(id, 25_000, 300);

        vm.prank(alice);
        vm.expectRevert(bytes("order not open"));
        vault.cancel(id);
    }

    function test_Cancel_TwiceReverts() public {
        uint256 id = _placeAliceOrder();

        vm.prank(alice);
        vault.cancel(id);

        vm.prank(alice);
        vm.expectRevert(bytes("order not open"));
        vault.cancel(id);
    }
}
