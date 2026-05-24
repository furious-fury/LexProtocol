// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {MarketContract} from "./MarketContract.sol";
import {PositionToken} from "./PositionToken.sol";
import {ResolutionEngine} from "./ResolutionEngine.sol";
import {Vault} from "./Vault.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {Ownable2Step} from "@openzeppelin/contracts/access/Ownable2Step.sol";

contract MarketFactory is Ownable2Step {
    PositionToken public immutable POSITION_TOKEN;
    Vault public immutable VAULT;
    ResolutionEngine public immutable RESOLUTION_ENGINE;

    uint256 public nextMarketId = 1;
    uint256 public maxPositionPerUser;
    uint256 public maxTotalMarketExposure;
    mapping(uint256 => address) public markets;
    mapping(address => bool) public isMarket;

    event MarketCreated(
        uint256 indexed marketId,
        address indexed market,
        address indexed creator,
        uint256 lockTime,
        string resolutionRule
    );
    event MarketLimitsUpdated(uint256 maxPositionPerUser, uint256 maxTotalMarketExposure);

    error ZeroAddress();
    error InvalidLockTime();

    constructor(address initialOwner, address positionToken_, address vault_, address resolutionEngine_)
        Ownable(initialOwner)
    {
        if (
            initialOwner == address(0) || positionToken_ == address(0) || vault_ == address(0)
                || resolutionEngine_ == address(0)
        ) revert ZeroAddress();

        POSITION_TOKEN = PositionToken(positionToken_);
        VAULT = Vault(payable(vault_));
        RESOLUTION_ENGINE = ResolutionEngine(resolutionEngine_);
    }

    function setMarketLimits(uint256 maxPositionPerUser_, uint256 maxTotalMarketExposure_) external onlyOwner {
        maxPositionPerUser = maxPositionPerUser_;
        maxTotalMarketExposure = maxTotalMarketExposure_;
        emit MarketLimitsUpdated(maxPositionPerUser_, maxTotalMarketExposure_);
    }

    function createMarket(string calldata resolutionRule, uint256 lockTime)
        external
        onlyOwner
        returns (address market)
    {
        if (lockTime <= block.timestamp) revert InvalidLockTime();

        uint256 marketId = nextMarketId++;
        market = address(
            new MarketContract(
                marketId,
                msg.sender,
                resolutionRule,
                lockTime,
                address(POSITION_TOKEN),
                address(VAULT),
                address(RESOLUTION_ENGINE),
                maxPositionPerUser,
                maxTotalMarketExposure
            )
        );

        markets[marketId] = market;
        isMarket[market] = true;

        POSITION_TOKEN.authorizeMarket(market, true);
        VAULT.authorizeMarket(market, true);
        RESOLUTION_ENGINE.authorizeMarket(market, true);

        emit MarketCreated(marketId, market, msg.sender, lockTime, resolutionRule);
    }

    function positionToken() external view returns (PositionToken) {
        return POSITION_TOKEN;
    }

    function vault() external view returns (Vault) {
        return VAULT;
    }

    function resolutionEngine() external view returns (ResolutionEngine) {
        return RESOLUTION_ENGINE;
    }
}
