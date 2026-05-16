// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {MarketContract} from "./MarketContract.sol";
import {PositionToken} from "./PositionToken.sol";
import {ResolutionEngine} from "./ResolutionEngine.sol";
import {Vault} from "./Vault.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

contract MarketFactory is Ownable {
    PositionToken public immutable POSITION_TOKEN;
    Vault public immutable VAULT;
    ResolutionEngine public immutable RESOLUTION_ENGINE;

    uint256 public nextMarketId = 1;
    mapping(uint256 => address) public markets;
    mapping(address => bool) public isMarket;

    event MarketCreated(
        uint256 indexed marketId,
        address indexed market,
        address indexed creator,
        uint256 lockTime,
        string resolutionRule
    );

    error InvalidLockTime();

    constructor(
        address initialOwner,
        address positionToken_,
        address vault_,
        address resolutionEngine_
    ) Ownable(initialOwner) {
        POSITION_TOKEN = PositionToken(positionToken_);
        VAULT = Vault(payable(vault_));
        RESOLUTION_ENGINE = ResolutionEngine(resolutionEngine_);
    }

    function createMarket(
        string calldata resolutionRule,
        uint256 lockTime
    ) external returns (address market) {
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
                address(RESOLUTION_ENGINE)
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
