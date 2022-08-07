// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;
// pragma abicoder v2;

import './ERC20PresetMinterPauser.sol';

/** 
 * @title Project
 * @dev Controls project permission management
 */
contract Project {
    address[] public maintainers;
    address private wallet;
    ERC20 private token;
    
    // TODO: Use to ensure that project wallet must always maintain a minimum of the
    // reserved tokens specified.
    uint256 private reservedTokens;
    
    // TODO: Add address of maintainer creating project.
    constructor(string memory _tokenName, string memory _tokenSymbol, uint256 _totalTokens, uint256 _reservedTokens) {
        token = new ERC20PresetMinterPauser(_tokenName, _tokenSymbol);
        reservedTokens = _reservedTokens;
    }

}