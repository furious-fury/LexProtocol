# LexProtocol Contracts

Foundry workspace for the LexProtocol truth layer.

## Build

```bash
forge build
```

## Test

```bash
forge test
```

## Local Deploy Dry Run

This compiles and executes the deploy script locally without broadcasting:

```bash
export PRIVATE_KEY="<deployer_private_key>"
export ORACLE_ADDRESS="<oracle_address>"

forge script script/Deploy.s.sol:Deploy
```

## Monad Testnet Deploy

Monad testnet uses chain ID `10143`. The contracts use native MON as collateral.

```bash
export PRIVATE_KEY="<deployer_private_key>"
export ORACLE_ADDRESS="<oracle_address>"

forge script script/Deploy.s.sol:Deploy \
  --rpc-url https://testnet-rpc.monad.xyz \
  --broadcast \
  --chain-id 10143
```

After deploying, clear the private key from the shell session:

```bash
unset PRIVATE_KEY
```

If your shell does not have `forge` on `PATH`, add Foundry first:

```bash
export PATH="$HOME/.foundry/bin:$PATH"
```
