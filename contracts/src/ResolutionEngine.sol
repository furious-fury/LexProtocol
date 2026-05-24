// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Constants} from "../lib/Constants.sol";
import {IOracle} from "../interfaces/IOracle.sol";
import {IResolutionEngine} from "../interfaces/IResolutionEngine.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {Ownable2Step} from "@openzeppelin/contracts/access/Ownable2Step.sol";

contract ResolutionEngine is Ownable2Step, IResolutionEngine {
    IOracle public oracleRegistry;
    address public factory;
    mapping(address => bool) public authorizedMarkets;
    mapping(uint256 => bool) public resolvedMarkets;

    event FactoryUpdated(address indexed factory);
    event MarketAuthorizationUpdated(address indexed market, bool authorized);
    event MarketResolved(uint256 indexed marketId, uint8 indexed outcome, uint256 indexed nonce);

    error NotAuthorized();
    error AlreadyResolved();
    error InvalidOutcome();
    error ZeroAddress();

    constructor(address initialOwner, address oracleRegistry_) Ownable(initialOwner) {
        if (initialOwner == address(0) || oracleRegistry_ == address(0)) revert ZeroAddress();
        oracleRegistry = IOracle(oracleRegistry_);
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

    function authorizeMarket(address market, bool authorized) external onlyOwnerOrFactory {
        if (market == address(0)) revert ZeroAddress();
        authorizedMarkets[market] = authorized;
        emit MarketAuthorizationUpdated(market, authorized);
    }

    function resolveMarket(uint256 marketId, bytes calldata oracleData, bytes calldata signature)
        external
        onlyMarket
        returns (uint8 outcome, uint256 nonce, uint256 expiry)
    {
        if (resolvedMarkets[marketId]) revert AlreadyResolved();
        (outcome, nonce, expiry) = oracleRegistry.validateSubmission(marketId, oracleData, signature);
        if (outcome != Constants.OUTCOME_YES && outcome != Constants.OUTCOME_NO) {
            revert InvalidOutcome();
        }

        resolvedMarkets[marketId] = true;
        emit MarketResolved(marketId, outcome, nonce);
    }

    function _onlyOwnerOrFactory() internal view {
        if (msg.sender != owner() && msg.sender != factory) revert NotAuthorized();
    }

    function _onlyMarket() internal view {
        if (!authorizedMarkets[msg.sender]) revert NotAuthorized();
    }
}
