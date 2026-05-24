// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {Ownable2Step} from "@openzeppelin/contracts/access/Ownable2Step.sol";
import {ERC1155} from "@openzeppelin/contracts/token/ERC1155/ERC1155.sol";
import {Constants} from "../lib/Constants.sol";

contract PositionToken is ERC1155, Ownable2Step {
    mapping(address => bool) public authorizedMarkets;

    address public factory;

    event FactoryUpdated(address indexed factory);
    event MarketAuthorizationUpdated(address indexed market, bool authorized);

    error NotAuthorized();
    error InvalidOutcome();
    error ZeroAddress();

    constructor(string memory uri_, address initialOwner) ERC1155(uri_) Ownable(initialOwner) {
        if (initialOwner == address(0)) revert ZeroAddress();
    }

    modifier onlyOwnerOrFactory() {
        _onlyOwnerOrFactory();
        _;
    }

    modifier onlyMarket() {
        _onlyMarket();
        _;
    }

    function setFactory(address factory_) external onlyOwner {
        if (factory_ == address(0)) revert ZeroAddress();
        factory = factory_;
        emit FactoryUpdated(factory_);
    }

    function setURI(string calldata uri_) external onlyOwner {
        _setURI(uri_);
    }

    function authorizeMarket(address market, bool authorized) external onlyOwnerOrFactory {
        if (market == address(0)) revert ZeroAddress();
        authorizedMarkets[market] = authorized;
        emit MarketAuthorizationUpdated(market, authorized);
    }

    function tokenId(uint256 marketId, uint8 outcome) public pure returns (uint256) {
        if (outcome != Constants.OUTCOME_YES && outcome != Constants.OUTCOME_NO) {
            revert InvalidOutcome();
        }
        return uint256(keccak256(abi.encodePacked("LEX_POSITION", marketId, outcome)));
    }

    function mint(address to, uint256 id, uint256 amount) external onlyMarket {
        _mint(to, id, amount, "");
    }

    function burn(address from, uint256 id, uint256 amount) external onlyMarket {
        _burn(from, id, amount);
    }

    function _onlyOwnerOrFactory() private view {
        if (msg.sender != owner() && msg.sender != factory) revert NotAuthorized();
    }

    function _onlyMarket() private view {
        if (!authorizedMarkets[msg.sender]) revert NotAuthorized();
    }
}
