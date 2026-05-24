# LexProtocol Security Model

This document summarizes the current security assumptions and hardening controls for the contracts, pricing service, and indexer.

## Trust Assumptions

- The protocol owner controls market creation through `MarketFactory`.
- The oracle signer is trusted to sign correct final outcomes.
- The oracle signer private key must never be committed to the repository or stored in `.env`.
- The pricing service is trusted only when it runs with the expected chain ID and oracle registry address.
- The indexer is an off-chain read model. Contracts remain the source of truth.

## Contract Controls

- Protocol admin contracts use two-step ownership transfer.
- Constructors and admin setters reject zero addresses.
- `MarketContract` and `Vault` use non-reentrancy guards around trading, resolution, redemption, and native MON payouts.
- Market contracts can enforce factory-configured per-user purchase caps and total market exposure caps.
- Markets emit `MarketFinalized` and move to `FINALIZED` once all winning position collateral is redeemed.
- Vault exposes `availableBalance()` and `reservedBalance()` so operators and tests can verify collateral accounting.

## Oracle And Replay Protection

- Oracle payloads use EIP-712 with:
  - name: `LexProtocol`
  - version: `1`
  - chain ID
  - verifying contract address
- `OracleRegistry` rejects:
  - wrong market IDs
  - invalid outcomes
  - expired submissions
  - zero nonces
  - reused nonces
  - invalid signatures
  - high-S signatures
  - invalid V values
- Nonces are globally unique on-chain.

## Pricing Service Controls

- `/price/:marketId` is informational and may be public.
- `/signed/:marketId` can be protected with a bearer token:

```bash
export PRICING_API_TOKEN="replace-with-a-strong-token"
```

- When `DATABASE_URL` is set, pricing nonces are allocated from Postgres and signed submissions are recorded in `pricing_nonces`.
- If `DATABASE_URL` is not set, the service falls back to in-memory nonces for local development only.
- Startup logs include signer address, chain ID, and oracle registry address, but never private keys.

## Indexer Controls

- `MarketCreated` is accepted only from the configured `MARKET_FACTORY_ADDRESS`.
- Market lifecycle events are accepted only from known market addresses created by that factory.
- Backfill is chunked with `INDEXER_BACKFILL_CHUNK_SIZE` to avoid unbounded RPC calls.
- Removed logs and canonical hash mismatches trigger rollback from the affected block.

## Private Key Handling

Use terminal exports for all private keys:

```bash
export PRIVATE_KEY="<deployer_private_key>"
export PRICING_SIGNER_PRIVATE_KEY="<oracle_signer_private_key>"
export E2E_DEPLOYER_PRIVATE_KEY="<factory_owner_private_key>"
export E2E_YES_TRADER_PRIVATE_KEY="<funded_yes_trader_private_key>"
export E2E_NO_TRADER_PRIVATE_KEY="<funded_no_trader_private_key>"
```

Clear keys when finished:

```bash
unset PRIVATE_KEY
unset PRICING_SIGNER_PRIVATE_KEY
unset E2E_DEPLOYER_PRIVATE_KEY
unset E2E_YES_TRADER_PRIVATE_KEY
unset E2E_NO_TRADER_PRIVATE_KEY
```

## Monad Deployment Checklist

- Confirm `CHAIN_ID=10143` for Monad testnet.
- Confirm `ORACLE_REGISTRY_ADDRESS` matches the deployed `OracleRegistry`.
- Confirm the pricing signer address equals `OracleRegistry.authorizedOracle`.
- Run migrations before starting the pricing/indexer services.
- Set `PRICING_API_TOKEN` before exposing the pricing service.
- Set `INDEXER_BACKFILL_CHUNK_SIZE` to a conservative value for the RPC provider.
- Run the full contract and Go test suites before deployment.

## Security Verification Commands

```bash
cd contracts && forge fmt
cd contracts && forge build
cd contracts && forge test
cd contracts && forge test --fuzz-runs 10000
go test ./backend/... ./shared/...
go build ./backend/... ./shared/...
```

Optional scanners:

```bash
slither contracts/src
gosec ./backend/... ./shared/...
```
