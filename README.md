# LexProtocol

EVM-based prediction market protocol with deterministic rule-based oracle resolution.

## Architecture

- **Truth Layer** — Solidity smart contracts (`contracts/`)
- **Pricing Layer** — Off-chain probability engine (`backend/cmd/pricing`)
- **Indexing Layer** — Event processor, Postgres, SSE (`backend/cmd/indexer`)

See [docs/lexprotocol_prd.md](docs/lexprotocol_prd.md) and [docs/lexprotocol_build_plan.md](docs/lexprotocol_build_plan.md).

## Phase 0 — Dev setup

### Prerequisites

- [Foundry](https://book.getfoundry.sh/getting-started/installation)
- Go 1.22+
- Docker

### Contracts

```bash
cd contracts
forge build
```

### Backend

```bash
# From repo root (uses go.work)
go build ./backend/...
go build ./shared/...
```

### Database

```bash
docker compose up -d postgres
```

Requires Docker Desktop to be running. Optional Redis: `docker compose --profile redis up -d`.

Copy `.env.example` to `.env` and adjust values if needed.

Verify DB connectivity (with Postgres running):

```bash
go test -tags=integration ./backend/internal/storage/...
```

### Indexer service

```bash
export DATABASE_URL="postgres://lexprotocol:lexprotocol@localhost:5432/lexprotocol?sslmode=disable"
export RPC_WS_URL="ws://127.0.0.1:8545"
export RPC_HTTP_URL="http://127.0.0.1:8545"
export MARKET_FACTORY_ADDRESS="<deployed_market_factory_address>"
export START_BLOCK="0"
export INDEXER_CONFIRMATIONS="2"
export INDEXER_BACKFILL_CHUNK_SIZE="2000"
export INDEXER_HTTP_ADDR=":8090"

go run ./backend/cmd/indexer
```

Then call:

```bash
curl http://localhost:8090/healthz
curl -N http://localhost:8090/events
```

The indexer stores pending logs, promotes confirmed logs to Postgres, writes normalized market/trade/oracle/settlement/redemption rows, and broadcasts confirmed events over SSE.
It only accepts `MarketCreated` events from the configured factory and only accepts market lifecycle events from known factory-created markets.

### Pricing service

The pricing service exposes informational prices and contract-compatible signed settlement payloads.

Do not store signer private keys in `.env` files. Export them only in the terminal session that runs the service:

```bash
export PRICING_SIGNER_PRIVATE_KEY="<oracle_signer_private_key>"
export ORACLE_REGISTRY_ADDRESS="<deployed_oracle_registry_address>"
export CHAIN_ID="10143"
export SIGNATURE_TTL_SECONDS="300"
export PRICING_HTTP_ADDR=":8080"
export DATABASE_URL="postgres://lexprotocol:lexprotocol@localhost:5432/lexprotocol?sslmode=disable"
export PRICING_API_TOKEN="<strong_local_token>"

go run ./backend/cmd/pricing
```

Then call:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/price/1
curl -H "Authorization: Bearer $PRICING_API_TOKEN" "http://localhost:8080/signed/1?outcome=YES"
```

`/price/:marketId` is informational only. `/signed/:marketId?outcome=YES|NO` returns `oracleData` and `signature` compatible with `MarketContract.submitOutcome`.
When `DATABASE_URL` is set, signed settlement nonces are persisted in Postgres so service restarts cannot reuse nonces. `PRICING_API_TOKEN` protects the signature endpoint; keep it unset only for isolated local development.

Clear the signer key when you are done:

```bash
unset PRICING_SIGNER_PRIVATE_KEY
unset PRICING_API_TOKEN
```

### Local end-to-end flow

Phase 4 is terminal-first and does not require a frontend. Run Postgres, Anvil, deploy contracts, start the pricing service and indexer, then execute the lifecycle command.

```bash
# Terminal 1
anvil
```

```bash
# Terminal 2
docker compose up -d postgres
```

Deploy contracts and copy the returned `OracleRegistry` and `MarketFactory` addresses:

```bash
cd contracts
export PRIVATE_KEY="<deployer_private_key>"
export ORACLE_ADDRESS="<oracle_signer_address>"
forge script script/Deploy.s.sol:Deploy \
  --rpc-url http://127.0.0.1:8545 \
  --broadcast
cd ..
```

Start pricing:

```bash
export PRICING_SIGNER_PRIVATE_KEY="<oracle_signer_private_key>"
export ORACLE_REGISTRY_ADDRESS="<deployed_oracle_registry_address>"
export CHAIN_ID="31337"
export DATABASE_URL="postgres://lexprotocol:lexprotocol@localhost:5432/lexprotocol?sslmode=disable"
export PRICING_API_TOKEN="<strong_local_token>"
go run ./backend/cmd/pricing
```

Start indexer:

```bash
export DATABASE_URL="postgres://lexprotocol:lexprotocol@localhost:5432/lexprotocol?sslmode=disable"
export RPC_WS_URL="ws://127.0.0.1:8545"
export RPC_HTTP_URL="http://127.0.0.1:8545"
export MARKET_FACTORY_ADDRESS="<deployed_market_factory_address>"
export START_BLOCK="0"
export INDEXER_CONFIRMATIONS="2"
export INDEXER_BACKFILL_CHUNK_SIZE="2000"
go run ./backend/cmd/indexer
```

Run the E2E lifecycle:

```bash
export RPC_HTTP_URL="http://127.0.0.1:8545"
export MARKET_FACTORY_ADDRESS="<deployed_market_factory_address>"
export E2E_DEPLOYER_PRIVATE_KEY="<factory_owner_private_key>"
export E2E_YES_TRADER_PRIVATE_KEY="<funded_yes_trader_private_key>"
export E2E_NO_TRADER_PRIVATE_KEY="<funded_no_trader_private_key>"
go run ./backend/cmd/e2e
```

The command creates a market, buys YES/NO, locks, fetches a signed YES outcome from pricing, resolves on-chain, redeems the winning YES position, and prints transaction hashes for indexer verification.

## Security

Phase 5 hardening adds reentrancy guards, zero-address validation, two-step ownership, configurable market exposure caps, market finalization events, persistent pricing nonces, protected signature generation, trusted indexer emitters, and chunked backfill.

See [docs/security_model.md](docs/security_model.md) and [docs/phase5_security_hardening_plan.md](docs/phase5_security_hardening_plan.md).
