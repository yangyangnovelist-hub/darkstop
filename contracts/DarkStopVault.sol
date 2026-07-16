// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

// TODO: Replace local interfaces with imports from flare-smart-contracts-v2 once published as a package.
import { ITeeExtensionRegistry } from "./interfaces/ITeeExtensionRegistry.sol";
import { ITeeMachineRegistry } from "./interfaces/ITeeMachineRegistry.sol";
import { TestFtsoV2Interface } from "flare-periphery/coston2/TestFtsoV2Interface.sol";

/// @notice Minimal ERC-20 surface needed for payouts.
interface IERC20 {
    function transfer(address to, uint256 amount) external returns (bool);
    function balanceOf(address account) external view returns (uint256);
}

/// @title DarkStopVault
/// @notice Confidential stop-loss vault on Flare. Users deposit C2FLR together
/// with an ECIES-encrypted trigger price; the ciphertext is forwarded to the
/// TEE extension via the FCC instruction flow. The chain only ever sees that
/// an order exists — the trigger price is revealed at settlement, where it is
/// re-verified against the live FTSO FLR/USD feed.
///
/// This contract absorbs the scaffold's InstructionSender role: it is the only
/// address allowed to submit instructions for the DarkStop extension.
///
/// DO NOT MODIFY (scaffold boilerplate): registry initialization in the
/// constructor, setExtensionId(), _getExtensionId(), and the
/// sendInstructions call path in _sendInstruction().
contract DarkStopVault {
    /// @notice Operation type for all DarkStop actions.
    // forge-lint: disable-next-line(unsafe-typecast)
    bytes32 public constant OP_TYPE_DARKSTOP = bytes32("DARKSTOP");

    /// @notice Command for placing a stop-loss order.
    // forge-lint: disable-next-line(unsafe-typecast)
    bytes32 public constant OP_COMMAND_PLACE = bytes32("PLACE_ORDER");

    /// @notice Command for cancelling an order.
    // forge-lint: disable-next-line(unsafe-typecast)
    bytes32 public constant OP_COMMAND_CANCEL = bytes32("CANCEL_ORDER");

    /// @notice FTSO v2 feed id for FLR/USD.
    bytes21 public constant FLR_USD = bytes21(0x01464c522f55534400000000000000000000000000);

    /// @notice Decimals of the payout token (USDT0).
    uint8 public constant PAYOUT_DECIMALS = 6;

    uint8 public constant STATUS_NONE = 0;
    uint8 public constant STATUS_OPEN = 1;
    uint8 public constant STATUS_EXECUTED = 2;
    uint8 public constant STATUS_CANCELLED = 3;

    /// @notice A stop-loss order. Deliberately contains NO price data:
    /// the trigger price lives encrypted inside the TEE only.
    struct Order {
        address owner;
        uint256 deposit;
        uint8 status;
    }

    // Instruction message layouts (mirrored by the Go extension's ABI decoders):
    //   PLACE_ORDER:  abi.encode(uint256 orderId, bytes ciphertext)
    //   CANCEL_ORDER: abi.encode(uint256 orderId)

    /// @notice Reference to the TEE extension registry contract.
    ITeeExtensionRegistry public immutable TEE_EXTENSION_REGISTRY;
    /// @notice Reference to the TEE machine registry contract.
    ITeeMachineRegistry public immutable TEE_MACHINE_REGISTRY;
    /// @notice FTSO v2 price feed reader (Coston2: ContractRegistry.getTestFtsoV2()).
    TestFtsoV2Interface public immutable FTSO_V2;
    /// @notice Testnet USDT0 used to pay out executed orders.
    IERC20 public immutable PAYOUT_TOKEN;

    /// @notice Contract owner (deployer): may configure the TEE executor.
    address public immutable OWNER;

    /// @notice Fee (in native token) forwarded to the registry per instruction.
    uint256 public immutable INSTRUCTION_FEE;

    /// @notice Orders by id. Ids start at 1.
    mapping(uint256 => Order) public orders;

    /// @notice Id of the most recently placed order (0 = none yet).
    uint256 public nextOrderId;

    /// @notice TEE settlement submitter, set by the owner after TEE registration.
    address public teeExecutor;

    uint256 private _extensionId;

    /// @notice Emitted when an order is placed. Carries NO price data.
    event OrderPlaced(uint256 indexed orderId, address indexed owner);

    /// @notice Emitted when an order is executed at the FTSO-verified price
    /// (USD per FLR, scaled to PAYOUT_DECIMALS).
    event OrderExecuted(uint256 indexed orderId, uint256 price);

    /// @notice Initializes the contract with registry addresses.
    /// @param _teeExtensionRegistry Address of the TEE extension registry.
    /// @param _teeMachineRegistry Address of the TEE machine registry.
    /// @param _ftsoV2 Address of the FTSO v2 feed reader.
    /// @param _payoutToken Address of the payout ERC-20 (6 decimals).
    /// @param _instructionFee Native fee forwarded per instruction.
    constructor(
        ITeeExtensionRegistry _teeExtensionRegistry,
        ITeeMachineRegistry _teeMachineRegistry,
        address _ftsoV2,
        address _payoutToken,
        uint256 _instructionFee
    ) {
        require(address(_teeExtensionRegistry) != address(0), "TeeExtensionRegistry cannot be zero address");
        require(address(_teeMachineRegistry) != address(0), "TeeMachineRegistry cannot be zero address");
        require(address(_teeExtensionRegistry).code.length > 0, "TeeExtensionRegistry has no code");
        require(address(_teeMachineRegistry).code.length > 0, "TeeMachineRegistry has no code");
        require(_ftsoV2 != address(0), "FtsoV2 cannot be zero address");
        require(_payoutToken != address(0), "Payout token cannot be zero address");
        TEE_EXTENSION_REGISTRY = _teeExtensionRegistry;
        TEE_MACHINE_REGISTRY = _teeMachineRegistry;
        FTSO_V2 = TestFtsoV2Interface(_ftsoV2);
        PAYOUT_TOKEN = IERC20(_payoutToken);
        OWNER = msg.sender;
        INSTRUCTION_FEE = _instructionFee;
    }

    /// @notice Finds and sets this contract's extension id. Can only be set once.
    /// DO NOT MODIFY this function.
    function setExtensionId() external {
        require(_extensionId == 0, "Extension ID already set.");

        uint256 c = TEE_EXTENSION_REGISTRY.extensionsCounter();
        for (uint256 i = 0; i < c; ++i) {
            if (TEE_EXTENSION_REGISTRY.getTeeExtensionInstructionsSender(i) == address(this)) {
                _extensionId = i;
                return;
            }
        }
        revert("Extension ID not found.");
    }

    /// @notice Places a confidential stop-loss order.
    /// @dev `msg.value` must exceed INSTRUCTION_FEE: the fee is forwarded to
    /// the registry with the PLACE_ORDER instruction, the remainder is held
    /// as the order's deposit (the C2FLR being protected).
    /// @param _ciphertext Trigger parameters, ECIES-encrypted to the TEE
    /// extension's public key. Opaque to this contract.
    /// @return id The new order's id.
    function placeOrder(bytes calldata _ciphertext) external payable returns (uint256 id) {
        require(_ciphertext.length > 0, "empty ciphertext");
        require(msg.value > INSTRUCTION_FEE, "value must exceed instruction fee");

        id = ++nextOrderId;
        orders[id] = Order({
            owner: msg.sender,
            deposit: msg.value - INSTRUCTION_FEE,
            status: STATUS_OPEN
        });
        emit OrderPlaced(id, msg.sender);

        _sendInstruction(OP_COMMAND_PLACE, abi.encode(id, _ciphertext), INSTRUCTION_FEE);
    }

    /// @notice Sets the TEE settlement submitter. Contract owner only.
    /// @param _teeExecutor Address allowed to call settle().
    function setTeeExecutor(address _teeExecutor) external {
        require(msg.sender == OWNER, "not contract owner");
        teeExecutor = _teeExecutor;
    }

    /// @notice Settles an order whose (TEE-revealed) trigger has been hit.
    /// @dev Only callable by the TEE executor. The contract does NOT trust the
    /// TEE alone: it re-reads the live FTSO FLR/USD feed and requires the
    /// current price to be fresh and at-or-below the revealed trigger.
    /// The deposit stays in the vault (testnet stand-in for the stable-side
    /// swap); the owner is paid `deposit * price` in USDT0.
    /// @param _orderId The order to settle.
    /// @param _triggerPrice Revealed trigger (USD per FLR, PAYOUT_DECIMALS scale).
    /// @param _maxAgeSec Maximum accepted FTSO price age in seconds.
    function settle(uint256 _orderId, uint256 _triggerPrice, uint256 _maxAgeSec) external {
        require(msg.sender == teeExecutor, "not tee executor");
        Order storage order = orders[_orderId];
        require(order.status == STATUS_OPEN, "order not open");

        (uint256 value, int8 decimals, uint64 timestamp) = FTSO_V2.getFeedById(FLR_USD);
        require(block.timestamp - timestamp <= _maxAgeSec, "stale price");

        uint256 price = _toPayoutDecimals(value, decimals);
        require(price <= _triggerPrice, "price above trigger");

        order.status = STATUS_EXECUTED;
        emit OrderExecuted(_orderId, price);

        // deposit is 18-decimals native token; price is PAYOUT_DECIMALS
        // USD/FLR, so the product / 1e18 is a PAYOUT_DECIMALS USD amount.
        uint256 payout = order.deposit * price / 1e18;
        require(PAYOUT_TOKEN.transfer(order.owner, payout), "payout transfer failed");
    }

    /// @notice Rescales an FTSO feed value (int8 decimals) to PAYOUT_DECIMALS.
    function _toPayoutDecimals(uint256 _value, int8 _decimals) internal pure returns (uint256) {
        int256 shift = int256(uint256(PAYOUT_DECIMALS)) - int256(_decimals);
        if (shift >= 0) {
            return _value * 10 ** uint256(shift);
        }
        return _value / 10 ** uint256(-shift);
    }

    /// @notice Sends an instruction to a random TEE machine of this extension.
    /// DO NOT MODIFY the sendInstructions call path.
    function _sendInstruction(bytes32 _opCommand, bytes memory _message, uint256 _fee) internal {
        address[] memory teeIds = TEE_MACHINE_REGISTRY.getRandomTeeIds(_getExtensionId(), 1);
        address[] memory cosigners = new address[](0);

        ITeeExtensionRegistry.TeeInstructionParams memory params = ITeeExtensionRegistry.TeeInstructionParams({
            opType: OP_TYPE_DARKSTOP,
            opCommand: _opCommand,
            message: _message,
            cosigners: cosigners,
            cosignersThreshold: 0,
            claimBackAddress: msg.sender
        });

        TEE_EXTENSION_REGISTRY.sendInstructions{value: _fee}(
            teeIds,
            params
        );
    }

    /// @notice Returns the cached extension ID, reverting if not yet set.
    /// @return The extension ID assigned to this contract.
    function _getExtensionId() internal view returns (uint256) {
        require(_extensionId != 0, "Extension ID is not set.");
        return _extensionId;
    }
}
