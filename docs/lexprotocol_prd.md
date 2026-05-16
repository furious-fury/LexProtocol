# LexProtocol — Prediction Market + Rule-Based Oracle Resolution (PRD)

## 1. System Overview

LexProtocol is an EVM-based prediction market protocol where markets are resolved deterministically using predefined authoritative rules rather than subjective consensus between multiple oracles.

The system is architected into three independent layers:

1. Truth Layer (On-chain Settlement)

* Determines final outcome
* Enforces deterministic resolution rules
* Handles collateral escrow and payout execution

2. Pricing Layer (Off-chain Go Service)

* Computes and updates market probabilities (informational odds only)
* Fetches external data sources (APIs)
* Serves cryptographically signed probability updates (EIP-712)
* Does NOT influence settlement or solvency

3. Indexing Layer (Backend Infrastructure)

* Tracks all market activity (Postgres)
* Stores trades, pricing history, oracle submissions, and settlement logs
* Provides real-time updates (SSE)
* Handles chain reorganization reconciliation and finality tracking

---

## 2. Core Design Philosophy

Truth is not derived from consensus or market disagreement. It is explicitly defined per market using deterministic rule constraints.

Each market defines:

* The authoritative resolution rule
* The validity window for resolution
* A single authorized oracle (MVP)

Pricing represents probabilistic expectation only and is explicitly separated from settlement logic.

Settlement is deterministic, atomic, and independent of pricing signals.

---

## 3. System Repository / Folder Structure

```
repo-root/

  contracts/
    src/
      MarketContract.sol
      MarketFactory.sol
      Vault.sol
      OracleRegistry.sol
      ResolutionEngine.sol
      PositionToken.sol

    interfaces/
      IMarket.sol
      IVault.sol
      IOracle.sol
      IResolutionEngine.sol

    lib/
      Math.sol
      Constants.sol

    test/
      Market.t.sol
      Vault.t.sol
      Resolution.t.sol

    script/
      Deploy.s.sol

  backend/
    cmd/
      indexer/
        main.go
      pricing/
        main.go

    internal/
      pricing/
        engine.go
        fetchers.go
        signer.go
        models.go

      indexer/
        listener.go
        processor.go
        reorg.go
        head_tracker.go
        broadcaster.go

      storage/
        postgres.go
        models.go

      api/
        server.go
        handlers.go

      events/
        types.go

  shared/
    types/
      market.go
      oracle.go
      events.go

  frontend/
    (optional)

  docs/
    lexprotocol_prd.md
    lexprotocol_build_plan.md
    architecture.md
```

---

## 4. Solidity Contract Interfaces

### IMarket.sol

```solidity
interface IMarket {
    function initialize(bytes calldata config) external;
    function buyYes(uint256 amount) external;
    function buyNo(uint256 amount) external;
    function lockMarket() external;
    function submitOutcome(bytes calldata oracleData, bytes calldata signature) external;
    function redeem() external;

    function getState() external view returns (uint8);
}
```

---

### IVault.sol (Liquidity Vault / House Model)

```solidity
interface IVault {
    function deposit(uint256 amount) external;
    function withdraw(uint256 amount) external;

    function coverExposure(uint256 marketId, uint256 amount) external;
    function settleExposure(uint256 marketId) external;

    function getBalance() external view returns (uint256);
}
```

---

### IOracle.sol

```solidity
interface IOracle {
    function submitUpdate(
        uint256 marketId,
        uint256 pYes,
        uint256 nonce,
        uint256 expiry,
        bytes calldata signature
    ) external;
}
```

---

### IResolutionEngine.sol

```solidity
interface IResolutionEngine {
    function resolveMarket(
        uint256 marketId,
        bytes calldata oracleData,
        bytes calldata signature
    ) external;
}
```

---

## 5. Go System Architecture

### Core Services

#### Indexer Service

* Listens to EVM logs
* Reconstructs state
* Handles reorgs
* Emits SSE events

#### Pricing Service

* Computes probabilities
* Calls external APIs
* Signs EIP-712 payloads
* Serves API + SSE updates

---

## 6. Go Internal Architecture

### Pricing Engine Flow

```
fetch market data → compute probability → sign payload → publish via API
```

### Worker Model

* goroutine 1: API fetchers
* goroutine 2: pricing computation
* goroutine 3: signing service
* goroutine 4: broadcaster

```

---

### Indexer Flow
```

RPC WS listener → pending_events → confirmation queue → Postgres → SSE broadcast

````

---

### Head Tracker
- tracks latest block
- validates canonical chain
- triggers reorg recovery

---

## 7. Event Schema (On-chain → Off-chain)

### TradeExecuted
```json
{
  "type": "TRADE_EXECUTED",
  "marketId": "uint256",
  "user": "address",
  "side": "YES|NO",
  "amount": "uint256",
  "blockNumber": "uint64",
  "txHash": "bytes32"
}
````

---

### MarketLocked

```json
{
  "type": "MARKET_LOCKED",
  "marketId": "uint256",
  "lockTime": "uint64",
  "blockNumber": "uint64"
}
```

---

### OracleSubmitted

```json
{
  "type": "ORACLE_SUBMITTED",
  "marketId": "uint256",
  "outcome": "YES|NO",
  "nonce": "uint256",
  "expiry": "uint64",
  "blockNumber": "uint64"
}
```

---

### MarketResolved

```json
{
  "type": "MARKET_RESOLVED",
  "marketId": "uint256",
  "outcome": "YES|NO",
  "blockNumber": "uint64"
}
```

---

## 8. Database Schema (Postgres)

### markets

* id
* creator
* status
* lock_time
* resolution_rule
* created_at

---

### trades

* id
* market_id
* user
* side
* amount
* tx_hash
* block_number

---

### oracle_updates

* id
* market_id
* p_yes
* nonce
* expiry
* signature
* created_at

---

### settlements

* id
* market_id
* outcome
* resolved_at

---

### blocks

* block_number
* block_hash
* parent_hash
* confirmed

---

## 9. SSE Event Stream Contract

### Channel Model

```
type Event struct {
    Type string
    Payload json.RawMessage
}
```

### Broadcaster Pattern

* map[clientID]chan Event
* fan-out goroutine per event
* auto-reconnect supported

---

## 10. State Machine (Implementation View)

CREATED → OPEN → LOCKED → RESOLVING → RESOLVED → FINALIZED
↘
INVALIDATED

---

## 11. Security Model

* EIP-712 domain separation (chainId + contract address)
* nonce-based replay protection
* expiry-based validity
* max position size per user
* deterministic lock enforcement

---

## 12. Key Implementation Rules

* Pricing layer NEVER writes to chain state
* Indexer only writes confirmed blocks
* Vault is sole counterparty for liquidity exposure
* Resolution is single-shot deterministic execution

---

## 13. MVP vs Advanced Roadmap

### MVP

* single oracle
* collateral mint/burn
* vault-backed liquidity
* Go pricing service
* deterministic resolution

---

### Phase 2

* bonding curve pricing
* improved UX odds engine

---

### Phase 3

* LMSR
* advanced market making

---

### Phase 4

* decentralized oracle network
* staking/slashing

---

## 14. Key Summary

LexProtocol is a deterministic settlement protocol with:

* on-chain truth enforcement
* off-chain probabilistic pricing
* vault-backed liquidity abstraction
* event-driven indexing architecture

It is designed for correctness first, scalability second, and decentralization as an evolutionary layer rather than a requirement at MVP stage.
