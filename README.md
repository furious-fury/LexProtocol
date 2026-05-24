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
export INDEXER_HTTP_ADDR=":8090"

go run ./backend/cmd/indexer
```

Then call:

```bash
curl http://localhost:8090/healthz
curl -N http://localhost:8090/events
```

The indexer stores pending logs, promotes confirmed logs to Postgres, writes normalized market/trade/oracle/settlement/redemption rows, and broadcasts confirmed events over SSE.

### Pricing service

The pricing service exposes informational prices and contract-compatible signed settlement payloads.

Do not store signer private keys in `.env` files. Export them only in the terminal session that runs the service:

```bash
export PRICING_SIGNER_PRIVATE_KEY="<oracle_signer_private_key>"
export ORACLE_REGISTRY_ADDRESS="<deployed_oracle_registry_address>"
export CHAIN_ID="10143"
export SIGNATURE_TTL_SECONDS="300"
export PRICING_HTTP_ADDR=":8080"

go run ./backend/cmd/pricing
```

Then call:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/price/1
curl "http://localhost:8080/signed/1?outcome=YES"
```

`/price/:marketId` is informational only. `/signed/:marketId?outcome=YES|NO` returns `oracleData` and `signature` compatible with `MarketContract.submitOutcome`.

Clear the signer key when you are done:

```bash
unset PRICING_SIGNER_PRIVATE_KEY
```
