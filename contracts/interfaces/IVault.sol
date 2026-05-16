// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IVault {
    function deposit() external payable;
    function withdraw(uint256 amount) external;
    function coverExposure(uint256 marketId, uint8 side) external payable;
    function settleExposure(uint256 marketId, uint8 outcome) external;
    function redeemPayout(uint256 marketId, address payable user, uint256 amount) external;
    function getBalance() external view returns (uint256);
}
