// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Constants} from "../lib/Constants.sol";
import {IMarket} from "../interfaces/IMarket.sol";
import {IResolutionEngine} from "../interfaces/IResolutionEngine.sol";
import {IVault} from "../interfaces/IVault.sol";
import {PositionToken} from "./PositionToken.sol";

contract MarketContract is IMarket {
    uint256 public immutable MARKET_ID;
    address public immutable CREATOR;
    PositionToken public immutable POSITION_TOKEN;
    IVault public immutable VAULT;
    IResolutionEngine public immutable RESOLUTION_ENGINE;

    string public resolutionRule;
    uint256 public lockTime;
    uint8 public status;
    uint8 public winningOutcome;

    uint256 public yesSupply;
    uint256 public noSupply;

    event TradeExecuted(
        uint256 indexed marketId,
        address indexed user,
        uint8 indexed side,
        uint256 amount
    );
    event MarketLocked(uint256 indexed marketId, uint256 lockTime);
    event OracleSubmitted(uint256 indexed marketId, uint8 indexed outcome, uint256 indexed nonce, uint256 expiry);
    event MarketResolved(uint256 indexed marketId, uint8 indexed outcome);
    event Redeemed(uint256 indexed marketId, address indexed user, uint256 amount);

    error MarketClosed();
    error MarketNotLockable();
    error MarketNotLocked();
    error MarketNotResolved();
    error AlreadyResolved();
    error InvalidAmount();
    error NothingToRedeem();

    constructor(
        uint256 marketId_,
        address creator_,
        string memory resolutionRule_,
        uint256 lockTime_,
        address positionToken_,
        address vault_,
        address resolutionEngine_
    ) {
        MARKET_ID = marketId_;
        CREATOR = creator_;
        resolutionRule = resolutionRule_;
        lockTime = lockTime_;
        POSITION_TOKEN = PositionToken(positionToken_);
        VAULT = IVault(vault_);
        RESOLUTION_ENGINE = IResolutionEngine(resolutionEngine_);
        status = Constants.STATUS_OPEN;
    }

    function buyYes() external payable {
        _buy(Constants.OUTCOME_YES);
    }

    function buyNo() external payable {
        _buy(Constants.OUTCOME_NO);
    }

    function lockMarket() external {
        if (status != Constants.STATUS_OPEN || block.timestamp < lockTime) revert MarketNotLockable();
        status = Constants.STATUS_LOCKED;
        emit MarketLocked(MARKET_ID, lockTime);
    }

    function submitOutcome(bytes calldata oracleData, bytes calldata signature) external {
        if (status == Constants.STATUS_OPEN && block.timestamp >= lockTime) {
            status = Constants.STATUS_LOCKED;
            emit MarketLocked(MARKET_ID, lockTime);
        }
        if (status != Constants.STATUS_LOCKED) revert MarketNotLocked();
        if (winningOutcome != Constants.OUTCOME_NONE) revert AlreadyResolved();

        status = Constants.STATUS_RESOLVING;
        (uint8 outcome, uint256 nonce, uint256 expiry) =
            RESOLUTION_ENGINE.resolveMarket(MARKET_ID, oracleData, signature);
        winningOutcome = outcome;
        VAULT.settleExposure(MARKET_ID, outcome);
        status = Constants.STATUS_RESOLVED;

        emit OracleSubmitted(MARKET_ID, outcome, nonce, expiry);
        emit MarketResolved(MARKET_ID, outcome);
    }

    function redeem() external {
        uint256 winningTokenId = POSITION_TOKEN.tokenId(MARKET_ID, winningOutcome);
        _redeem(POSITION_TOKEN.balanceOf(msg.sender, winningTokenId));
    }

    function redeem(uint256 amount) external {
        _redeem(amount);
    }

    function _redeem(uint256 amount) private {
        if (status != Constants.STATUS_RESOLVED && status != Constants.STATUS_FINALIZED) {
            revert MarketNotResolved();
        }

        uint256 winningTokenId = POSITION_TOKEN.tokenId(MARKET_ID, winningOutcome);
        if (amount == 0) revert NothingToRedeem();

        POSITION_TOKEN.burn(msg.sender, winningTokenId, amount);
        VAULT.redeemPayout(MARKET_ID, payable(msg.sender), amount);
        emit Redeemed(MARKET_ID, msg.sender, amount);
    }

    function getState() external view returns (uint8) {
        return status;
    }

    function marketId() external view returns (uint256) {
        return MARKET_ID;
    }

    function creator() external view returns (address) {
        return CREATOR;
    }

    function positionToken() external view returns (PositionToken) {
        return POSITION_TOKEN;
    }

    function vault() external view returns (IVault) {
        return VAULT;
    }

    function resolutionEngine() external view returns (IResolutionEngine) {
        return RESOLUTION_ENGINE;
    }

    function _buy(uint8 side) private {
        if (status != Constants.STATUS_OPEN || block.timestamp >= lockTime) revert MarketClosed();
        if (msg.value == 0) revert InvalidAmount();

        uint256 id = POSITION_TOKEN.tokenId(MARKET_ID, side);
        POSITION_TOKEN.mint(msg.sender, id, msg.value);
        VAULT.coverExposure{value: msg.value}(MARKET_ID, side);

        if (side == Constants.OUTCOME_YES) {
            yesSupply += msg.value;
        } else {
            noSupply += msg.value;
        }

        emit TradeExecuted(MARKET_ID, msg.sender, side, msg.value);
    }
}
