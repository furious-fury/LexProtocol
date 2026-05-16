# LexProtocol

EVM-based prediction market protocol with deterministic rule-based oracle resolution.

## Architecture

- **Truth Layer** — Solidity smart contracts (`contracts/`)
- **Pricing Layer** — Off-chain probability engine (`backend/cmd/pricing`)
- **Indexing Layer** — Event processor, Postgres, SSE (`backend/cmd/indexer`)

See [docs/lexprotocol_prd.md](docs/lexprotocol_prd.md) and [docs/lexprotocol_build_plan.md](docs/lexprotocol_build_plan.md).

Track progress in [tasklist.md](tasklist.md).

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

### Run services (stubs)

```bash
go run ./backend/cmd/pricing
go run ./backend/cmd/indexer
```
