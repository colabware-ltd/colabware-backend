// SPDX-License-Identifier: MIT
pragma solidity ^0.8.11;
// pragma abicoder v2;

/** 
 * @title Project
 * @dev Controls project permission management
 */
contract Project {
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
    
    
}

interface IERC20 {

    function totalSupply() external view returns (uint256);
    function balanceOf(address account) external view returns (uint256);
    function allowance(address owner, address spender) external view returns (uint256);

    function transfer(address recipient, uint256 amount) external returns (bool);
    function approve(address spender, uint256 amount) external returns (bool);
    function transferFrom(address sender, address recipient, uint256 amount) external returns (bool);


    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
}

contract Token is IERC20 {

    string public name;
    string public symbol;
    uint8 public constant decimals = 18;


    mapping(address => uint256) balances;

    mapping(address => mapping (address => uint256)) allowed;

    uint256 totalSupply_;


   constructor(string memory _name, string memory _symbol, uint256 _totalSupply) {
    name = _name;
    symbol = _symbol;
    totalSupply_ = _totalSupply;
   }

    function totalSupply() public override view returns (uint256) {
        return totalSupply_;
    }

    function balanceOf(address tokenOwner) public override view returns (uint256) {
        return balances[tokenOwner];
    }

    function transfer(address receiver, uint256 numTokens) public override returns (bool) {
        require(numTokens <= balances[msg.sender]);
        balances[msg.sender] = balances[msg.sender]-numTokens;
        balances[receiver] = balances[receiver]+numTokens;
        emit Transfer(msg.sender, receiver, numTokens);
        return true;
    }

    function approve(address delegate, uint256 numTokens) public override returns (bool) {
        allowed[msg.sender][delegate] = numTokens;
        emit Approval(msg.sender, delegate, numTokens);
        return true;
    }

    function allowance(address owner, address delegate) public override view returns (uint) {
        return allowed[owner][delegate];
    }

    function transferFrom(address owner, address buyer, uint256 numTokens) public override returns (bool) {
        require(numTokens <= balances[owner]);
        require(numTokens <= allowed[owner][msg.sender]);

        balances[owner] = balances[owner]-numTokens;
        allowed[owner][msg.sender] = allowed[owner][msg.sender]-numTokens;
        balances[buyer] = balances[buyer]+numTokens;
        emit Transfer(owner, buyer, numTokens);
        return true;
    }
}
