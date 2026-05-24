// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Constants} from "../lib/Constants.sol";
import {IVault} from "../interfaces/IVault.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {Ownable2Step} from "@openzeppelin/contracts/access/Ownable2Step.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

contract Vault is Ownable2Step, ReentrancyGuard, IVault {
    struct MarketExposure {
        uint256 yesCollateral;
        uint256 noCollateral;
        uint8 winningOutcome;
        bool settled;
    }

    mapping(address => bool) public authorizedMarkets;
    mapping(uint256 => MarketExposure) public exposures;

    address public factory;
    uint256 public reservedCollateral;

    event Deposited(address indexed depositor, uint256 amount);
    event Withdrawn(address indexed recipient, uint256 amount);
    event FactoryUpdated(address indexed factory);
    event MarketAuthorizationUpdated(address indexed market, bool authorized);
    event ExposureCovered(uint256 indexed marketId, uint8 indexed side, uint256 amount);
    event ExposureSettled(uint256 indexed marketId, uint8 indexed outcome);
    event PayoutRedeemed(uint256 indexed marketId, address indexed user, uint256 amount);

    error NotAuthorized();
    error InvalidOutcome();
    error MarketAlreadySettled();
    error MarketNotSettled();
    error InsufficientAvailableBalance();
    error TransferFailed();
    error ZeroAddress();
    error InvalidAmount();

    constructor(address initialOwner) Ownable(initialOwner) {
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

    receive() external payable {
        emit Deposited(msg.sender, msg.value);
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

    function deposit() external payable {
        if (msg.value == 0) revert InvalidAmount();
        emit Deposited(msg.sender, msg.value);
    }

    function withdraw(uint256 amount) external onlyOwner nonReentrant {
        if (amount == 0) revert InvalidAmount();
        if (address(this).balance - reservedCollateral < amount) revert InsufficientAvailableBalance();
        address currentOwner = owner();
        (bool ok,) = payable(currentOwner).call{value: amount}("");
        if (!ok) revert TransferFailed();
        emit Withdrawn(currentOwner, amount);
    }

    function coverExposure(uint256 marketId, uint8 side) external payable onlyMarket {
        if (msg.value == 0) revert InvalidAmount();
        if (side != Constants.OUTCOME_YES && side != Constants.OUTCOME_NO) revert InvalidOutcome();
        MarketExposure storage exposure = exposures[marketId];
        if (exposure.settled) revert MarketAlreadySettled();

        if (side == Constants.OUTCOME_YES) {
            exposure.yesCollateral += msg.value;
        } else {
            exposure.noCollateral += msg.value;
        }

        reservedCollateral += msg.value;
        emit ExposureCovered(marketId, side, msg.value);
    }

    function settleExposure(uint256 marketId, uint8 outcome) external onlyMarket {
        if (outcome != Constants.OUTCOME_YES && outcome != Constants.OUTCOME_NO) revert InvalidOutcome();
        MarketExposure storage exposure = exposures[marketId];
        if (exposure.settled) revert MarketAlreadySettled();

        exposure.winningOutcome = outcome;
        exposure.settled = true;
        if (outcome == Constants.OUTCOME_YES) {
            reservedCollateral -= exposure.noCollateral;
        } else {
            reservedCollateral -= exposure.yesCollateral;
        }
        emit ExposureSettled(marketId, outcome);
    }

    function redeemPayout(uint256 marketId, address payable user, uint256 amount) external onlyMarket nonReentrant {
        if (user == address(0)) revert ZeroAddress();
        if (amount == 0) revert InvalidAmount();
        MarketExposure storage exposure = exposures[marketId];
        if (!exposure.settled) revert MarketNotSettled();
        reservedCollateral -= amount;

        (bool ok,) = user.call{value: amount}("");
        if (!ok) revert TransferFailed();
        emit PayoutRedeemed(marketId, user, amount);
    }

    function getBalance() external view returns (uint256) {
        return address(this).balance;
    }

    function availableBalance() external view returns (uint256) {
        return address(this).balance - reservedCollateral;
    }

    function reservedBalance() external view returns (uint256) {
        return reservedCollateral;
    }

    function _onlyOwnerOrFactory() internal view {
        if (msg.sender != owner() && msg.sender != factory) revert NotAuthorized();
    }

    function _onlyMarket() internal view {
        if (!authorizedMarkets[msg.sender]) revert NotAuthorized();
    }
}
