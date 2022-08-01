// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;
// pragma abicoder v2;

import './Token.sol';

/** 
 * @title Project
 * @dev Controls project permission management
 */
contract Project {
    address[] public maintainers;
    TokenEntry[] public tokens;

    struct TokenEntry {
        string tokenName;
        address tokenAddress;
    }

    constructor(string memory _tokenName, string memory _tokenSymbol, uint256 _tokenSupply) {
        createToken(_tokenName, _tokenSymbol, _tokenSupply);
    }

    /** 
     * @dev Create a new token.
     * @param _name name of the token
     * @param _symbol symbol of the token
     * @param _supply total supply of tokens
     */
    function createToken(string memory _name, string memory _symbol, uint256 _supply) 
        public
    {
        address newTokenAddress = address(
            new Token(_name, _symbol, _supply)
        );

        tokens.push();
        uint index = tokens.length - 1;

        tokens[index].tokenName = _name;
        tokens[index].tokenAddress = newTokenAddress;
    }
    
    function getTokens() public view returns (TokenEntry[] memory) {
        return tokens;
    }

    // TODO: Create function for purchasing token that transfers amount
    // to maintainers with each transfer to investor.
    
}
