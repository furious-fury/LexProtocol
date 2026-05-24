// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {Constants} from "../lib/Constants.sol";
import {MarketContract} from "../src/MarketContract.sol";
import {MarketFactory} from "../src/MarketFactory.sol";
import {OracleRegistry} from "../src/OracleRegistry.sol";
import {PositionToken} from "../src/PositionToken.sol";
import {ResolutionEngine} from "../src/ResolutionEngine.sol";
import {Vault} from "../src/Vault.sol";

contract MarketTest is Test {
    uint256 private oracleKey = 0xA11CE;
    address private oracle = vm.addr(oracleKey);
    address private alice = address(0xA11CE);
    address private bob = address(0xB0B);
    address private charlie = address(0xCA11);

    PositionToken private positionToken;
    Vault private vault;
    OracleRegistry private oracleRegistry;
    ResolutionEngine private resolutionEngine;
    MarketFactory private factory;

    function setUp() public {
        positionToken = new PositionToken("ipfs://positions/{id}.json", address(this));
        vault = new Vault(address(this));
        oracleRegistry = new OracleRegistry(address(this), oracle);
        resolutionEngine = new ResolutionEngine(address(this), address(oracleRegistry));
        factory = new MarketFactory(address(this), address(positionToken), address(vault), address(resolutionEngine));

        positionToken.setFactory(address(factory));
        vault.setFactory(address(factory));
        resolutionEngine.setFactory(address(factory));

        vm.deal(alice, 10 ether);
        vm.deal(bob, 10 ether);
        vm.deal(charlie, 10 ether);
    }

    function testCreateMarketAndBuyTokenizedPositions() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        uint256 yesId = positionToken.tokenId(market.marketId(), Constants.OUTCOME_YES);
        assertEq(positionToken.balanceOf(alice, yesId), 1 ether);
        assertEq(address(vault).balance, 1 ether);
        assertEq(market.yesSupply(), 1 ether);
    }

    function testPositionTokensAreTransferable() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        uint256 yesId = positionToken.tokenId(market.marketId(), Constants.OUTCOME_YES);

        vm.prank(alice);
        positionToken.safeTransferFrom(alice, bob, yesId, 0.4 ether, "");

        assertEq(positionToken.balanceOf(alice, yesId), 0.6 ether);
        assertEq(positionToken.balanceOf(bob, yesId), 0.4 ether);
    }

    function testResolutionAndWinningRedemption() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        vm.prank(bob);
        market.buyNo{value: 2 ether}();

        vm.warp(market.lockTime());
        market.lockMarket();

        (bytes memory oracleData, bytes memory signature) =
            _signedSubmission(market.marketId(), Constants.OUTCOME_YES, 1, block.timestamp + 1 days);

        market.submitOutcome(oracleData, signature);
        assertEq(market.winningOutcome(), Constants.OUTCOME_YES);
        assertEq(market.getState(), Constants.STATUS_RESOLVED);

        uint256 aliceBefore = alice.balance;
        vm.prank(alice);
        market.redeem();

        uint256 yesId = positionToken.tokenId(market.marketId(), Constants.OUTCOME_YES);
        assertEq(positionToken.balanceOf(alice, yesId), 0);
        assertEq(alice.balance, aliceBefore + 1 ether);
        assertEq(market.getState(), Constants.STATUS_FINALIZED);

        vm.prank(bob);
        vm.expectRevert();
        market.redeem();
    }

    function testInvalidOracleSignatureFails() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        vm.warp(market.lockTime());
        market.lockMarket();

        bytes memory oracleData = abi.encode(
            OracleRegistry.OracleSubmission({
                marketId: market.marketId(), outcome: Constants.OUTCOME_YES, nonce: 1, expiry: block.timestamp + 1 days
            })
        );
        bytes32 digest = _submissionDigest(market.marketId(), Constants.OUTCOME_YES, 1, block.timestamp + 1 days);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(0xBAD, digest);
        bytes memory signature = abi.encodePacked(r, s, v);

        vm.expectRevert();
        market.submitOutcome(oracleData, signature);
    }

    function testOracleNonceReplayFails() public {
        MarketContract marketA = _createMarket();
        MarketContract marketB = _createMarket();

        vm.prank(alice);
        marketA.buyYes{value: 1 ether}();
        vm.prank(alice);
        marketB.buyYes{value: 1 ether}();

        vm.warp(marketA.lockTime());
        marketA.lockMarket();
        marketB.lockMarket();

        (bytes memory dataA, bytes memory sigA) =
            _signedSubmission(marketA.marketId(), Constants.OUTCOME_YES, 1, block.timestamp + 1 days);
        marketA.submitOutcome(dataA, sigA);

        (bytes memory dataB, bytes memory sigB) =
            _signedSubmission(marketB.marketId(), Constants.OUTCOME_YES, 1, block.timestamp + 1 days);
        vm.expectRevert();
        marketB.submitOutcome(dataB, sigB);
    }

    function testUnauthorizedPositionMintAndBurnFail() public {
        MarketContract market = _createMarket();
        uint256 yesId = positionToken.tokenId(market.marketId(), Constants.OUTCOME_YES);

        vm.prank(alice);
        vm.expectRevert();
        positionToken.mint(alice, yesId, 1 ether);

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        vm.prank(alice);
        vm.expectRevert();
        positionToken.burn(alice, yesId, 1 ether);
    }

    function testVaultCannotWithdrawReservedCollateral() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        vm.expectRevert();
        vault.withdraw(1 ether);
        assertEq(vault.reservedBalance(), 1 ether);
        assertEq(vault.availableBalance(), 0);
    }

    function testExpiredOracleSubmissionFails() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        vm.warp(market.lockTime());
        market.lockMarket();

        (bytes memory oracleData, bytes memory signature) =
            _signedSubmission(market.marketId(), Constants.OUTCOME_YES, 2, block.timestamp - 1);

        vm.expectRevert();
        market.submitOutcome(oracleData, signature);
    }

    function testWrongMarketOraclePayloadFails() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        vm.warp(market.lockTime());
        market.lockMarket();

        (bytes memory oracleData, bytes memory signature) =
            _signedSubmission(market.marketId() + 1, Constants.OUTCOME_YES, 2, block.timestamp + 1 days);

        vm.expectRevert();
        market.submitOutcome(oracleData, signature);
    }

    function testTransferredWinningTokenCanRedeem() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        uint256 yesId = positionToken.tokenId(market.marketId(), Constants.OUTCOME_YES);
        vm.prank(alice);
        positionToken.safeTransferFrom(alice, bob, yesId, 0.4 ether, "");

        _resolve(market, Constants.OUTCOME_YES, 2);

        uint256 bobBefore = bob.balance;
        vm.prank(bob);
        market.redeem();

        assertEq(positionToken.balanceOf(bob, yesId), 0);
        assertEq(bob.balance, bobBefore + 0.4 ether);
    }

    function testPartialRedemption() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        uint256 yesId = positionToken.tokenId(market.marketId(), Constants.OUTCOME_YES);
        _resolve(market, Constants.OUTCOME_YES, 2);

        uint256 aliceBefore = alice.balance;
        vm.prank(alice);
        market.redeem(0.25 ether);

        assertEq(positionToken.balanceOf(alice, yesId), 0.75 ether);
        assertEq(alice.balance, aliceBefore + 0.25 ether);
    }

    function testDuplicateResolutionFails() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        _resolve(market, Constants.OUTCOME_YES, 2);

        (bytes memory oracleData, bytes memory signature) =
            _signedSubmission(market.marketId(), Constants.OUTCOME_YES, 3, block.timestamp + 1 days);

        vm.expectRevert();
        market.submitOutcome(oracleData, signature);
    }

    function _createMarket() private returns (MarketContract) {
        address market = factory.createMarket("BTC closes above 100000 USD by lock time", block.timestamp + 1 days);
        return MarketContract(payable(market));
    }

    function _resolve(MarketContract market, uint8 outcome, uint256 nonce) private {
        vm.warp(market.lockTime());
        market.lockMarket();

        (bytes memory oracleData, bytes memory signature) =
            _signedSubmission(market.marketId(), outcome, nonce, block.timestamp + 1 days);
        market.submitOutcome(oracleData, signature);
    }

    function _signedSubmission(uint256 marketId, uint8 outcome, uint256 nonce, uint256 expiry)
        private
        view
        returns (bytes memory oracleData, bytes memory signature)
    {
        oracleData = abi.encode(
            OracleRegistry.OracleSubmission({marketId: marketId, outcome: outcome, nonce: nonce, expiry: expiry})
        );
        bytes32 digest = _submissionDigest(marketId, outcome, nonce, expiry);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(oracleKey, digest);
        signature = abi.encodePacked(r, s, v);
    }

    function _submissionDigest(uint256 marketId, uint8 outcome, uint256 nonce, uint256 expiry)
        private
        view
        returns (bytes32)
    {
        bytes32 structHash =
            keccak256(abi.encode(oracleRegistry.ORACLE_SUBMISSION_TYPEHASH(), marketId, outcome, nonce, expiry));
        return keccak256(abi.encodePacked("\x19\x01", oracleRegistry.domainSeparator(), structHash));
    }

    // ── Access control tests ──────────────────────────────────────────────

    function testNonOwnerCannotCreateMarket() public {
        vm.prank(alice);
        vm.expectRevert(abi.encodeWithSelector(Ownable.OwnableUnauthorizedAccount.selector, alice));
        factory.createMarket("Unauthorized market", block.timestamp + 1 days);
    }

    function testOwnerCanCreateMarket() public {
        // address(this) is the owner set in setUp
        address market = factory.createMarket("Owner-created market", block.timestamp + 1 days);
        assertTrue(market != address(0));
        assertTrue(factory.isMarket(market));
    }

    function testPerUserPositionLimit() public {
        factory.setMarketLimits(1 ether, 0);
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 0.6 ether}();

        vm.prank(alice);
        vm.expectRevert(MarketContract.ExceedsPositionLimit.selector);
        market.buyYes{value: 0.5 ether}();
    }

    function testMarketExposureLimit() public {
        factory.setMarketLimits(0, 1 ether);
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 0.7 ether}();

        vm.prank(bob);
        vm.expectRevert(MarketContract.ExceedsMarketLimit.selector);
        market.buyNo{value: 0.4 ether}();
    }

    function testZeroAddressAdminUpdatesFail() public {
        vm.expectRevert();
        positionToken.setFactory(address(0));

        vm.expectRevert();
        vault.setFactory(address(0));

        vm.expectRevert();
        resolutionEngine.setFactory(address(0));

        vm.expectRevert();
        oracleRegistry.setAuthorizedOracle(address(0));
    }

    function testOracleZeroNonceFails() public {
        MarketContract market = _createMarket();

        vm.prank(alice);
        market.buyYes{value: 1 ether}();

        vm.warp(market.lockTime());
        market.lockMarket();

        (bytes memory oracleData, bytes memory signature) =
            _signedSubmission(market.marketId(), Constants.OUTCOME_YES, 0, block.timestamp + 1 days);

        vm.expectRevert(OracleRegistry.InvalidNonce.selector);
        market.submitOutcome(oracleData, signature);
    }
}
