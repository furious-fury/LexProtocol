// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Constants} from "../lib/Constants.sol";
import {IOracle} from "../interfaces/IOracle.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {Ownable2Step} from "@openzeppelin/contracts/access/Ownable2Step.sol";

contract OracleRegistry is Ownable2Step, IOracle {
    struct OracleSubmission {
        uint256 marketId;
        uint8 outcome;
        uint256 nonce;
        uint256 expiry;
    }

    bytes32 public constant ORACLE_SUBMISSION_TYPEHASH =
        keccak256("OracleSubmission(uint256 marketId,uint8 outcome,uint256 nonce,uint256 expiry)");
    bytes32 private constant DOMAIN_TYPEHASH =
        keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)");
    bytes32 private constant NAME_HASH = keccak256("LexProtocol");
    bytes32 private constant VERSION_HASH = keccak256("1");

    address public authorizedOracle;
    mapping(uint256 => bool) public usedNonces;

    event AuthorizedOracleUpdated(address indexed oracle);
    event OracleSubmissionValidated(uint256 indexed marketId, uint8 indexed outcome, uint256 indexed nonce);

    error InvalidOracle();
    error InvalidOutcome();
    error InvalidMarket();
    error ExpiredSubmission();
    error NonceAlreadyUsed();
    error InvalidSignature();
    error InvalidSignatureLength();
    error InvalidSignatureS();
    error InvalidSignatureV();
    error ZeroAddress();
    error InvalidNonce();

    constructor(address initialOwner, address initialOracle) Ownable(initialOwner) {
        if (initialOwner == address(0) || initialOracle == address(0)) revert ZeroAddress();
        authorizedOracle = initialOracle;
        emit AuthorizedOracleUpdated(initialOracle);
    }

    function setAuthorizedOracle(address oracle) external onlyOwner {
        if (oracle == address(0)) revert ZeroAddress();
        authorizedOracle = oracle;
        emit AuthorizedOracleUpdated(oracle);
    }

    function domainSeparator() public view returns (bytes32 result) {
        bytes32 typeHash = DOMAIN_TYPEHASH;
        bytes32 nameHash = NAME_HASH;
        bytes32 versionHash = VERSION_HASH;
        uint256 chainId = block.chainid;
        address verifyingContract = address(this);

        assembly {
            let ptr := mload(0x40)
            mstore(ptr, typeHash)
            mstore(add(ptr, 0x20), nameHash)
            mstore(add(ptr, 0x40), versionHash)
            mstore(add(ptr, 0x60), chainId)
            mstore(add(ptr, 0x80), verifyingContract)
            result := keccak256(ptr, 0xa0)
        }
    }

    function validateSubmission(uint256 expectedMarketId, bytes calldata oracleData, bytes calldata signature)
        external
        returns (uint8 outcome, uint256 nonce, uint256 expiry)
    {
        OracleSubmission memory submission = abi.decode(oracleData, (OracleSubmission));
        if (submission.marketId != expectedMarketId) revert InvalidMarket();
        if (submission.outcome != Constants.OUTCOME_YES && submission.outcome != Constants.OUTCOME_NO) {
            revert InvalidOutcome();
        }
        if (block.timestamp > submission.expiry) revert ExpiredSubmission();
        if (submission.nonce == 0) revert InvalidNonce();
        if (usedNonces[submission.nonce]) revert NonceAlreadyUsed();

        bytes32 structHash = _submissionHash(submission);
        bytes32 digest = _typedDataHash(domainSeparator(), structHash);
        address signer = _recover(digest, signature);
        if (signer != authorizedOracle || signer == address(0)) revert InvalidSignature();

        usedNonces[submission.nonce] = true;
        emit OracleSubmissionValidated(submission.marketId, submission.outcome, submission.nonce);

        return (submission.outcome, submission.nonce, submission.expiry);
    }

    function _submissionHash(OracleSubmission memory submission) private pure returns (bytes32 result) {
        bytes32 typeHash = ORACLE_SUBMISSION_TYPEHASH;
        assembly {
            let ptr := mload(0x40)
            mstore(ptr, typeHash)
            mstore(add(ptr, 0x20), mload(submission))
            mstore(add(ptr, 0x40), mload(add(submission, 0x20)))
            mstore(add(ptr, 0x60), mload(add(submission, 0x40)))
            mstore(add(ptr, 0x80), mload(add(submission, 0x60)))
            result := keccak256(ptr, 0xa0)
        }
    }

    function _typedDataHash(bytes32 domain, bytes32 structHash) private pure returns (bytes32 result) {
        assembly {
            let ptr := mload(0x40)
            mstore(ptr, 0x1901000000000000000000000000000000000000000000000000000000000000)
            mstore(add(ptr, 0x02), domain)
            mstore(add(ptr, 0x22), structHash)
            result := keccak256(ptr, 0x42)
        }
    }

    function _recover(bytes32 digest, bytes calldata signature) private pure returns (address) {
        if (signature.length != 65) revert InvalidSignatureLength();

        bytes32 r;
        bytes32 s;
        uint8 v;
        assembly {
            r := calldataload(signature.offset)
            s := calldataload(add(signature.offset, 32))
            v := byte(0, calldataload(add(signature.offset, 64)))
        }

        if (v < 27) v += 27;
        if (v != 27 && v != 28) revert InvalidSignatureV();
        if (uint256(s) > 0x7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0) {
            revert InvalidSignatureS();
        }
        return ecrecover(digest, v, r, s);
    }
}
