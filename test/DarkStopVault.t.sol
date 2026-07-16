// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import { Test } from "forge-std/Test.sol";
import { DarkStopVault } from "../contracts/DarkStopVault.sol";
import { MockUSDT0 } from "../contracts/MockUSDT0.sol";
import { ITeeExtensionRegistry } from "../contracts/interfaces/ITeeExtensionRegistry.sol";
import { ITeeMachineRegistry } from "../contracts/interfaces/ITeeMachineRegistry.sol";

/// @notice Mock of the TEE extension registry: records the last instruction it
/// received so tests can assert on opType/opCommand/message/fee.
contract MockTeeExtensionRegistry is ITeeExtensionRegistry {
    address public registeredSender;

    uint256 public sendCount;
    bytes32 public lastOpType;
    bytes32 public lastOpCommand;
    bytes public lastMessage;
    uint256 public lastValue;
    address public lastClaimBackAddress;
    uint256 public lastTeeIdCount;

    function setRegisteredSender(address _sender) external {
        registeredSender = _sender;
    }

    function sendInstructions(
        address[] calldata _teeIds,
        TeeInstructionParams calldata _instructionParams
    ) external payable returns (bytes32 _instructionId) {
        sendCount++;
        lastTeeIdCount = _teeIds.length;
        lastOpType = _instructionParams.opType;
        lastOpCommand = _instructionParams.opCommand;
        lastMessage = _instructionParams.message;
        lastValue = msg.value;
        lastClaimBackAddress = _instructionParams.claimBackAddress;
        return keccak256(abi.encode(sendCount));
    }

    function extensionsCounter() external pure returns (uint256) {
        return 2;
    }

    /// @dev The vault is registered at extension id 1 (id 0 is unusable:
    /// the scaffold's setExtensionId() treats 0 as "unset").
    function getTeeExtensionInstructionsSender(uint256 _extensionId) external view returns (address) {
        return _extensionId == 1 ? registeredSender : address(0);
    }
}

/// @notice Mock of the TEE machine registry: returns one fixed TEE id.
contract MockTeeMachineRegistry is ITeeMachineRegistry {
    function getRandomTeeIds(uint256, uint256 _count) external pure returns (address[] memory) {
        address[] memory ids = new address[](_count);
        for (uint256 i = 0; i < _count; i++) {
            ids[i] = address(uint160(0x7EE) + uint160(i));
        }
        return ids;
    }
}

/// @notice Mock FTSO v2: test-settable (value, decimals, timestamp) for FLR/USD.
contract MockFtsoV2 {
    uint256 public value;
    int8 public decimals;
    uint64 public timestamp;

    function setFeed(uint256 _value, int8 _decimals, uint64 _timestamp) external {
        value = _value;
        decimals = _decimals;
        timestamp = _timestamp;
    }

    function getFeedById(bytes21) external view returns (uint256, int8, uint64) {
        return (value, decimals, timestamp);
    }
}

contract DarkStopVaultTest is Test {
    event OrderPlaced(uint256 indexed orderId, address indexed owner);

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
}
