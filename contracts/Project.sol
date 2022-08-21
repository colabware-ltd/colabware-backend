// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;
// pragma abicoder v2;

import './ERC20PresetMinterPauser.sol';

/** 
 * @title Project
 * @dev Controls project permission management
 */
contract Project {
    address[] private maintainers;
    address private wallet;
    ERC20PresetMinterPauser private token;
    
    // TODO: Use to ensure that project wallet must always maintain a minimum of the
    // reserved tokens specified (i.e. when selling)
    uint256 private reservedTokens;
    
    // TODO: Add address of maintainer creating project.
    constructor(string memory _tokenName, string memory _tokenSymbol, uint256 _totalTokens, uint256 _reservedTokens, address _wallet) {
        token = new ERC20PresetMinterPauser(_tokenName, _tokenSymbol);
        token.mint(_wallet, _totalTokens);
        reservedTokens = _reservedTokens;
        wallet = _wallet;
    }

    function getReservedTokens() public view returns (uint256) {
        return reservedTokens;
    }

    function getTokenSupply() public view returns (uint256) {
        return token.totalSupply();
    }

    function getBalance(address account) public view returns (uint256) {
        return token.balanceOf(account);
    }

    function listBalances() public view returns (uint256, uint256, uint256) {
        return (this.getBalance(wallet), getReservedTokens(), this.getTokenSupply() - this.getBalance(wallet));
    }
}