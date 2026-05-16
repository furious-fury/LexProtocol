// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IMarket {
    function buyYes() external payable;
    function buyNo() external payable;
    function lockMarket() external;
    function submitOutcome(bytes calldata oracleData, bytes calldata signature) external;
    function redeem() external;
    function redeem(uint256 amount) external;
    function getState() external view returns (uint8);
}
