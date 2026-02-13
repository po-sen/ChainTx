// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract MockUSDT {
    string public constant name = "Tether USD";
    string public constant symbol = "USDT";
    uint8 public constant decimals = 6;

    uint256 public totalSupply;
    mapping(address => uint256) public balanceOf;

    event Transfer(address indexed from, address indexed to, uint256 value);

    constructor(address initialHolder, uint256 initialAmount) {
        require(initialHolder != address(0), "initial holder is required");
        _mint(initialHolder, initialAmount);
    }

    function transfer(address to, uint256 amount) external returns (bool) {
        require(to != address(0), "recipient is required");

        uint256 senderBalance = balanceOf[msg.sender];
        require(senderBalance >= amount, "insufficient balance");

        unchecked {
            balanceOf[msg.sender] = senderBalance - amount;
        }
        balanceOf[to] += amount;

        emit Transfer(msg.sender, to, amount);
        return true;
    }

    function _mint(address to, uint256 amount) internal {
        totalSupply += amount;
        balanceOf[to] += amount;
        emit Transfer(address(0), to, amount);
    }
}
