// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IOracle {
    function validateSubmission(
        uint256 expectedMarketId,
        bytes calldata oracleData,
        bytes calldata signature
    ) external returns (uint8 outcome, uint256 nonce, uint256 expiry);
}
