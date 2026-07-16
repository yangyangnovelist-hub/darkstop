// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import { ITeeExtensionRegistry } from "../interfaces/ITeeExtensionRegistry.sol";
import { ITeeMachineRegistry } from "../interfaces/ITeeMachineRegistry.sol";

// Mock TEE/FTSO dependencies shared by the Foundry unit tests and the local
// dev stack (scripts/dev-stack.sh → contracts/dev/DevStack.s.sol).
// NEVER deploy these to a public network.

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
