// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script} from "forge-std/Script.sol";
import {MarketFactory} from "../src/MarketFactory.sol";
import {OracleRegistry} from "../src/OracleRegistry.sol";
import {PositionToken} from "../src/PositionToken.sol";
import {ResolutionEngine} from "../src/ResolutionEngine.sol";
import {Vault} from "../src/Vault.sol";

contract Deploy is Script {
    function run() external returns (
        PositionToken positionToken,
        Vault vault,
        OracleRegistry oracleRegistry,
        ResolutionEngine resolutionEngine,
        MarketFactory marketFactory
    ) {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        address oracle = vm.envAddress("ORACLE_ADDRESS");

        vm.startBroadcast(deployerPrivateKey);

        address deployer = vm.addr(deployerPrivateKey);
        positionToken = new PositionToken("https://lexprotocol.io/positions/{id}.json", deployer);
        vault = new Vault(deployer);
        oracleRegistry = new OracleRegistry(deployer, oracle);
        resolutionEngine = new ResolutionEngine(deployer, address(oracleRegistry));
        marketFactory = new MarketFactory(
            deployer,
            address(positionToken),
            address(vault),
            address(resolutionEngine)
        );

        positionToken.setFactory(address(marketFactory));
        vault.setFactory(address(marketFactory));
        resolutionEngine.setFactory(address(marketFactory));

        vm.stopBroadcast();
    }
}
