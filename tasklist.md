# LexProtocol — Master Tasklist

Track implementation progress. Check items off as you complete them (`- [x]`).

---

## Phase 0: Project Setup

### Repository initialization
- [x] Create `contracts/` directory tree (`src`, `interfaces`, `lib`, `test`, `script`)
- [x] Create `backend/` directory tree (`cmd`, `internal`)
- [x] Create `shared/types/` directory
- [x] Create `docs/` and move PRD + build plan
- [x] Create `frontend/` placeholder

### Tooling
- [x] Initialize Foundry in `contracts/`
- [x] Pin `solc_version` in `foundry.toml`
- [x] Verify `forge build` succeeds
- [x] Initialize Go module in `shared/`
- [x] Initialize Go module in `backend/`
- [x] Add root `go.work`
- [x] Add `docker-compose.yml` for Postgres
- [x] Add optional Redis profile in Docker Compose
- [x] Add `.gitignore`, `.env.example`, `README.md`

### Shared types
- [x] Define `MarketStatus` and `MarketState` in `shared/types/market.go`
- [x] Define `Outcome` and `OracleSubmission` in `shared/types/oracle.go`
- [x] Define `TradeEvent`, `Event`, and event constants in `shared/types/events.go`
- [x] Backend stubs import shared types (`cmd/pricing`, `cmd/indexer`)
- [x] `backend/internal/storage/postgres.go` — Open + Ping helper

### Phase 0 verification
- [x] `forge build` in `contracts/`
- [x] `go build ./backend/...` and `go build ./shared/...` from repo root
- [ ] `docker compose up -d postgres` and DB reachable (start Docker Desktop, then run)
- [ ] Integration test: `go test -tags=integration ./backend/internal/storage/...`

---

## Phase 1: Core Smart Contracts (Truth Layer MVP)

### MarketContract
- [x] Implement `createMarket()` (via `MarketFactory`)
- [x] Implement `buyYes()` / `buyNo()`
- [x] Implement `lockMarket()`
- [x] Implement `submitOutcome()`
- [x] Implement `redeem()`
- [x] Add `IMarket.sol` interface

### Vault
- [x] Implement `deposit()` / `withdraw()`
- [x] Implement `coverExposure()` / `settleExposure()`
- [x] Add `IVault.sol` interface

### OracleRegistry
- [x] Store authorized oracle address
- [x] Validate EIP-712 signatures
- [x] Add `IOracle.sol` interface

### ResolutionEngine
- [x] Validate oracle submission
- [x] Finalize outcome and trigger payout
- [x] Add `IResolutionEngine.sol` interface

### Supporting contracts
- [x] `MarketFactory.sol`
- [x] `PositionToken.sol`
- [x] `Math.sol` / `Constants.sol` libs (`Constants.sol`; no math helper needed yet)

### Tests & deploy
- [x] Unit tests: market lifecycle
- [x] Unit tests: mint/burn logic
- [x] Unit tests: resolution correctness
- [x] Deploy script (`script/Deploy.s.sol`)
- [x] Deployable testnet build

### Phase 1 stabilization
- [x] Use OpenZeppelin ERC1155 for tokenized YES/NO positions
- [x] Use OpenZeppelin Ownable for protocol admin contracts
- [x] Add tests for unauthorized position mint/burn
- [x] Add tests for vault reserved-collateral withdrawal protection
- [x] Add tests for expired oracle submissions
- [x] Add tests for wrong-market oracle payloads
- [x] Add tests for transferred winning-token redemption
- [x] Add tests for partial redemption
- [x] Add tests for duplicate resolution attempts
- [x] Add Monad testnet deployment notes

---

## Phase 2: Pricing Service (Go Backend)

### Workers
- [ ] API Fetcher worker
- [x] Probability computation worker (static MVP engine)
- [x] EIP-712 signing worker (settlement outcome payloads)
- [x] API server worker

### Components
- [ ] External API fetchers (crypto, sports, events)
- [x] Pricing engine — compute P(YES)
- [ ] Pricing engine — volatility spread
- [x] Signing service (marketId, outcome, nonce, expiry)
- [x] `GET /price/:marketId`
- [x] `GET /signed/:marketId`
- [x] `GET /healthz`
- [x] In-memory global nonce store
- [x] Config validation for chain ID, oracle registry, signer key, and signature TTL
- [x] Tests for config, nonce, EIP-712 signing, ABI payload encoding, and HTTP handlers

### Deliverable
- [x] Running Go service returning informational prices and signed settlement payloads

---

## Phase 2.5: Integration Hardening

### Go ↔ Solidity compatibility
- [x] Foundry `SignatureCompat.t.sol` — proves `OracleRegistry.validateSubmission` accepts Go-signed EIP-712 payloads
- [x] Go `compat_test.go` — digest determinism, struct hash, ABI encoding cross-check
- [x] Low-V signature test (Go's `crypto.Sign` returns V=0/1, contract normalizes)

### Contract access control
- [x] Add `onlyOwner` to `MarketFactory.createMarket()` (admin-only market creation)
- [x] Test: non-owner rejected with `OwnableUnauthorizedAccount`
- [x] Test: owner can still create markets
- [x] Full access control audit — all other functions already guarded

### Documentation
- [x] `docs/pricing_service_guide.md` — CLI quickstart, curl examples, env reference
- [x] Persistent nonce storage plan — Postgres schema + migration path documented

### Architecture decision
- [x] Admin UI will call contracts directly via wallet (standard dApp pattern, no backend relay)
- [x] `onlyOwner` on `createMarket()` enforces access on-chain

---

## Phase 3: Indexer Service

### Components
- [x] RPC WebSocket listener
- [x] Block confirmation worker (configurable confirmations, default: 2)
- [x] Reorg handler (canonical block hash mismatch + removed-log rollback)
- [x] Postgres writer (`markets`, `trades`, `oracle_submissions`, `settlements`, `redemptions`, `blocks`)
- [x] SSE broadcaster
- [x] SQL migrations for indexer tables
- [x] Indexer config validation

### Event flow
- [x] Listen to contract logs → `pending_events`
- [x] Confirm blocks → `confirmed_events`
- [x] Broadcast events via SSE

### Deliverable
- [x] Fully functional event indexer
- [x] SSE stream consumable by frontend

---

## Phase 4: End-to-End Integration

- [ ] Connect MarketContract to indexer
- [ ] Feed pricing data to frontend via API
- [ ] Verify oracle submission flow (off-chain sign → on-chain resolve)
- [ ] Test lifecycle: CREATED → OPEN → LOCKED → RESOLVED → FINALIZED

---

## Phase 5: Security Hardening

- [ ] EIP-712 replay protection validation
- [ ] Enforce nonce + expiry rules
- [ ] Max position size per user
- [ ] Vault collateral consistency checks
- [ ] Finality confirmation checks in indexer

---

## Phase 6: MVP Frontend (Optional)

- [ ] Market list page
- [ ] Market detail page
- [ ] Trade interface (YES/NO)
- [ ] Live price feed via SSE

---

## MVP Completion Criteria

- [ ] Users can create and trade YES/NO positions
- [ ] Market locks deterministically
- [ ] Oracle submits final outcome
- [ ] Resolution executed on-chain
- [ ] Indexer reflects final state correctly
- [ ] SSE updates frontend in real-time

---

## Post-MVP (Not required)

- [ ] LMSR pricing engine
- [ ] Multi-oracle consensus
- [ ] Staking / slashing system
- [ ] Permissionless oracle registry
- [ ] Fully decentralized pricing feeds
