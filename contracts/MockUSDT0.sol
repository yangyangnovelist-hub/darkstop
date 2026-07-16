// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

/// @title MockUSDT0
/// @notice Minimal 6-decimals ERC-20 used as the testnet payout token for
/// DarkStopVault. The deployer (owner) can mint a payout pool to the vault.
contract MockUSDT0 {
    string public constant name = "Mock USDT0";
    string public constant symbol = "USDT0";
    uint8 public constant decimals = 6;

    address public immutable owner;
    uint256 public totalSupply;

    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);

    constructor() {
        owner = msg.sender;
    }

    /// @notice Mints `_amount` tokens to `_to`. Owner only.
    function mint(address _to, uint256 _amount) external {
        require(msg.sender == owner, "not owner");
        totalSupply += _amount;
        balanceOf[_to] += _amount;
        emit Transfer(address(0), _to, _amount);
    }

    function transfer(address _to, uint256 _amount) external returns (bool) {
        return _transfer(msg.sender, _to, _amount);
    }

    function transferFrom(address _from, address _to, uint256 _amount) external returns (bool) {
        uint256 allowed = allowance[_from][msg.sender];
        require(allowed >= _amount, "insufficient allowance");
        if (allowed != type(uint256).max) {
            allowance[_from][msg.sender] = allowed - _amount;
        }
        return _transfer(_from, _to, _amount);
    }

    function approve(address _spender, uint256 _amount) external returns (bool) {
        allowance[msg.sender][_spender] = _amount;
        emit Approval(msg.sender, _spender, _amount);
        return true;
    }

    function _transfer(address _from, address _to, uint256 _amount) internal returns (bool) {
        require(balanceOf[_from] >= _amount, "insufficient balance");
        balanceOf[_from] -= _amount;
        balanceOf[_to] += _amount;
        emit Transfer(_from, _to, _amount);
        return true;
    }
}
