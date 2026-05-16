# Step-by-Step Build Plan: LexProtocol

## Overview

This document translates the PRD into an actionable implementation roadmap for building the MVP of LexProtocol. It is structured for a single-developer execution flow with clear sequencing across smart contracts, backend services, and indexing infrastructure.

The system is divided into three layers:
- Truth Layer (Solidity smart contracts)
- Pricing Layer (Go backend service)
- Indexing Layer (Go event processor + Postgres + SSE)

---

## Phase 0: Project Setup (Day 1)

### Objectives
- Initialize monorepo
- Define shared types
- Set up development tooling

### Tasks

#### Repository Initialization
- Create monorepo structure:
  - contracts/
  - backend/
  - shared/
  - docs/

#### Tooling
- Solidity: Foundry or Hardhat
- Go modules initialized in backend/
- Postgres instance (local Docker or native install)
- Redis optional (for SSE buffering)

#### Shared Types
Define initial shared structs:
- MarketState
- OracleSubmission
- TradeEvent

---

## Phase 1: Core Smart Contracts (Truth Layer MVP)

### Objectives
Implement minimal deterministic settlement system.

### Contracts to Build

#### 1. MarketContract
Responsibilities:
- Create market
- Lock market
- Handle YES/NO minting
- Store collateral
- Trigger resolution

Key functions:
- createMarket()
- buyYes()
- buyNo()
- lockMarket()
- submitOutcome()
- redeem()

---

#### 2. Vault Contract (Liquidity Backing)
Responsibilities:
- Hold collateral
- Ensure solvency
- Act as counterparty for pricing abstraction

Key functions:
- deposit()
- withdraw()
- coverExposure()
- settleExposure()

---

#### 3. OracleRegistry
Responsibilities:
- Store authorized oracle address
- Validate EIP-712 signatures

---

#### 4. ResolutionEngine
Responsibilities:
- Validate oracle submission
- Finalize outcome
- Trigger payout logic

---

### Deliverable
- Deployable testnet contracts
- Unit tests for:
  - market lifecycle
  - mint/burn logic
  - resolution correctness

---

## Phase 2: Pricing Service (Go Backend)

### Objectives
Build off-chain probability engine.

### Architecture

Workers:
- API Fetcher Worker
- Probability Computation Worker
- EIP-712 Signing Worker
- API Server Worker

---

### Components

#### 1. Fetchers
- Pull external APIs (crypto, sports, events)

#### 2. Pricing Engine
- Compute P(YES)
- Apply volatility spread

#### 3. Signing Service
- Generate EIP-712 signed payloads
- Include:
  - marketId
  - pYes
  - nonce
  - expiry

#### 4. API Layer
- GET /price/:marketId
- GET /signed/:marketId

---

### Deliverable
- Running Go service returning signed probability updates

---

## Phase 3: Indexer Service

### Objectives
Track all on-chain events and maintain off-chain state.

### Architecture

Components:
- RPC Listener (WebSocket)
- Block Confirmation Worker
- Reorg Handler
- Postgres Writer
- SSE Broadcaster

---

### Event Flow

1. Listen to contract logs
2. Write to pending_events
3. Wait N confirmations (default: 5–12 blocks)
4. Move to confirmed_events
5. Broadcast via SSE

---

### Reorg Handling
- Track block hash per height
- Detect mismatch
- Roll back affected blocks
- Re-sync from last stable block

---

### Deliverable
- Fully functional event indexer
- SSE stream to frontend

---

## Phase 4: End-to-End Integration

### Objectives
Connect contracts, backend, and indexer.

### Tasks

- Connect MarketContract to indexer
- Feed pricing data into frontend via API
- Verify oracle submission flow
- Test full lifecycle:
  CREATED → OPEN → LOCKED → RESOLVED → FINALIZED

---

## Phase 5: Security Hardening

### Objectives
Ensure LexProtocol safety and correctness.

### Tasks

- Add EIP-712 replay protection validation
- Enforce nonce + expiry rules
- Add max position size per user
- Validate vault collateral consistency
- Add finality confirmation checks in indexer

---

## Phase 6: MVP Frontend (Optional but Recommended)

### Objectives
Minimal UI for demonstration.

### Pages
- Market list
- Market detail
- Trade interface
- Live price feed (SSE)

---

## MVP Completion Criteria

The system is considered complete when:

- Users can create and trade YES/NO positions
- Market locks deterministically
- Oracle submits final outcome
- Resolution is executed on-chain
- Indexer reflects final state correctly
- SSE updates frontend in real-time

---

## Post-MVP Extensions (Not Required)

- LMSR pricing engine
- Multi-oracle consensus
- Staking/slashing system
- Permissionless oracle registry
- Fully decentralized pricing feeds

---

## Final Notes

The MVP prioritizes:

- Deterministic settlement correctness
- Minimal trust assumptions
- Clean separation of system layers
- Debuggable and observable state transitions
