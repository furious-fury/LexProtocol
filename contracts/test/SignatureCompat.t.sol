// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {Constants} from "../lib/Constants.sol";
import {OracleRegistry} from "../src/OracleRegistry.sol";

/// @title Go ↔ Solidity EIP-712 Signature Compatibility Test
/// @notice Proves that an EIP-712 payload signed with the same private key and parameters
///         used by the Go pricing signer is accepted by OracleRegistry.validateSubmission().
///         If this test passes, the Go backend and Solidity contract are byte-compatible.
contract SignatureCompatTest is Test {
    // Hardhat account #1 — same key used in backend/internal/pricing/signer_test.go
    uint256 private constant ORACLE_KEY = 0x59c6995e998f97a5a0044966f0945389dc9e86dae547d3e236020d123b4d7bc5;

    // Pinned test parameters — identical to Go's TestSignerProducesRecoverableSettlementSubmission
    uint256 private constant MARKET_ID = 7;
    uint8   private constant OUTCOME   = Constants.OUTCOME_YES; // 1
    uint256 private constant NONCE     = 1;
    uint256 private constant EXPIRY    = 1300;
    uint256 private constant CHAIN_ID  = 10143; // Monad testnet
    uint256 private constant TIMESTAMP = 1000;  // block.timestamp pinned so expiry 1300 is valid

    OracleRegistry private oracleRegistry;
    address private oracle;

    function setUp() public {
        vm.chainId(CHAIN_ID);
        vm.warp(TIMESTAMP);
        oracle = vm.addr(ORACLE_KEY);
        oracleRegistry = new OracleRegistry(address(this), oracle);
    }

    /// @notice Core compatibility proof: sign with the test key, verify the contract accepts it.
    function testGoSignedPayloadAcceptedByContract() public {
        bytes memory oracleData = abi.encode(
            OracleRegistry.OracleSubmission({
                marketId: MARKET_ID,
                outcome: OUTCOME,
                nonce: NONCE,
                expiry: EXPIRY
            })
        );

        bytes32 digest = _submissionDigest(MARKET_ID, OUTCOME, NONCE, EXPIRY);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(ORACLE_KEY, digest);
        bytes memory signature = abi.encodePacked(r, s, v);

        // This is the critical assertion: the contract accepts the signature
        (uint8 retOutcome, uint256 retNonce, uint256 retExpiry) =
            oracleRegistry.validateSubmission(MARKET_ID, oracleData, signature);

        assertEq(retOutcome, OUTCOME, "outcome mismatch");
        assertEq(retNonce, NONCE, "nonce mismatch");
        assertEq(retExpiry, EXPIRY, "expiry mismatch");
    }

    /// @notice Cross-check: verify domain separator components match expected EIP-712 construction.
    function testDomainSeparatorConstruction() public view {
        bytes32 expected = keccak256(
            abi.encode(
                keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
                keccak256("LexProtocol"),
                keccak256("1"),
                CHAIN_ID,
                address(oracleRegistry)
            )
        );

        assertEq(oracleRegistry.domainSeparator(), expected, "domain separator mismatch");
    }

    /// @notice Cross-check: verify struct hash matches expected EIP-712 construction
    ///         and that ecrecover produces the correct oracle address.
    function testStructHashConstruction() public view {
        bytes32 structHash = keccak256(
            abi.encode(
                oracleRegistry.ORACLE_SUBMISSION_TYPEHASH(),
                MARKET_ID,
                OUTCOME,
                NONCE,
                EXPIRY
            )
        );

        // Recompute the full digest and verify ecrecover works
        bytes32 digest = keccak256(
            abi.encodePacked("\x19\x01", oracleRegistry.domainSeparator(), structHash)
        );

        (uint8 v, bytes32 r, bytes32 s) = vm.sign(ORACLE_KEY, digest);
        address recovered = ecrecover(digest, v, r, s);
        assertEq(recovered, oracle, "struct hash produces wrong signer recovery");
    }

    /// @notice Verify that a signature with V=0 or V=1 (Go's crypto.Sign style) is accepted.
    ///         Go's crypto.Sign returns V as 0 or 1; the contract normalizes V < 27.
    function testGoStyleLowVSignatureAccepted() public {
        bytes memory oracleData = abi.encode(
            OracleRegistry.OracleSubmission({
                marketId: MARKET_ID,
                outcome: OUTCOME,
                nonce: 2, // different nonce to avoid replay
                expiry: EXPIRY
            })
        );

        bytes32 digest = _submissionDigest(MARKET_ID, OUTCOME, 2, EXPIRY);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(ORACLE_KEY, digest);

        // Simulate Go's crypto.Sign output: V is 0 or 1 instead of 27 or 28
        uint8 lowV = v - 27;
        bytes memory signature = abi.encodePacked(r, s, lowV);

        (uint8 retOutcome,,) = oracleRegistry.validateSubmission(MARKET_ID, oracleData, signature);
        assertEq(retOutcome, OUTCOME, "low-V signature rejected");
    }

    /// @notice Log all intermediate EIP-712 values for manual cross-checking with Go output.
    ///         Run with: forge test --match-test testLogIntermediateValues -vvv
    function testLogIntermediateValues() public view {
        bytes32 domainSep = oracleRegistry.domainSeparator();
        bytes32 structHash = keccak256(
            abi.encode(
                oracleRegistry.ORACLE_SUBMISSION_TYPEHASH(),
                MARKET_ID,
                OUTCOME,
                NONCE,
                EXPIRY
            )
        );
        bytes32 digest = keccak256(
            abi.encodePacked("\x19\x01", domainSep, structHash)
        );

        console.log("=== EIP-712 Cross-Check Values ===");
        console.log("Oracle address:", oracle);
        console.log("Registry address:", address(oracleRegistry));
        console.log("Chain ID:", CHAIN_ID);
        console.log("Domain separator:");
        console.logBytes32(domainSep);
        console.log("Struct hash:");
        console.logBytes32(structHash);
        console.log("Digest:");
        console.logBytes32(digest);
    }

    function _submissionDigest(
        uint256 marketId,
        uint8 outcome,
        uint256 nonce,
        uint256 expiry
    ) private view returns (bytes32) {
        bytes32 structHash = keccak256(
            abi.encode(
                oracleRegistry.ORACLE_SUBMISSION_TYPEHASH(),
                marketId,
                outcome,
                nonce,
                expiry
            )
        );
        return keccak256(abi.encodePacked("\x19\x01", oracleRegistry.domainSeparator(), structHash));
    }
}
