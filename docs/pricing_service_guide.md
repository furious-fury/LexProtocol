# Pricing Service — CLI Guide

How to run the pricing service locally and interact with it.

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.26+ | Backend runtime |
| Foundry | Latest | Contract compilation, deployment, testing |
| Docker | Latest | Postgres (required for persistent nonce storage) |

---

## Quick Start

### 1. Export secrets

> **Never put private keys in `.env` files.** Export them in your terminal session only.

```bash
# The private key of the authorized oracle signer (the address registered in OracleRegistry)
export PRICING_SIGNER_PRIVATE_KEY="0x<your-oracle-signer-key>"

# Address of the deployed OracleRegistry contract
export ORACLE_REGISTRY_ADDRESS="0x<deployed-registry>"

# Chain ID (Monad testnet = 10143, local Anvil = 31337)
export CHAIN_ID="10143"
```

### 2. Start the service

```bash
go run ./backend/cmd/pricing
```

You should see:

```
lexprotocol pricing service listening on :8080
```

### 3. Test the endpoints

```bash
# Health check
curl http://localhost:8080/healthz

# Informational price quote (static 50/50 for MVP)
curl http://localhost:8080/price/1

# Signed settlement payload — YES outcome
curl "http://localhost:8080/signed/1?outcome=YES"

# Signed settlement payload — NO outcome
curl "http://localhost:8080/signed/1?outcome=NO"
```

---

## Response Format

### `GET /healthz`

```json
{"status": "ok"}
```

### `GET /price/:marketId`

```json
{
  "marketId": 1,
  "pYes": 500000,
  "pNo": 500000,
  "confidence": "stub",
  "source": "static",
  "asOf": "2026-05-16T19:00:00Z"
}
```

Probabilities are scaled by `1,000,000`. A value of `500000` = 50%.

### `GET /signed/:marketId?outcome=YES`

```json
{
  "marketId": 1,
  "outcome": "YES",
  "outcomeId": 1,
  "nonce": 1,
  "expiry": 1747422300,
  "oracleData": "0x0000...0001...0001...<128 bytes hex>",
  "signature": "0x<65 bytes hex>",
  "signer": "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
  "domain": {
    "name": "LexProtocol",
    "version": "1",
    "chainId": 10143,
    "verifyingContract": "0x<OracleRegistry address>"
  }
}
```

The `oracleData` and `signature` fields are the exact values to pass to
`MarketContract.submitOutcome(oracleData, signature)`.

---

## Submitting On-Chain

### Using `cast` (Foundry)

```bash
# Take the oracleData and signature from the /signed response
ORACLE_DATA="0x<from response>"
SIGNATURE="0x<from response>"
MARKET_ADDRESS="0x<deployed MarketContract>"

cast send $MARKET_ADDRESS \
  "submitOutcome(bytes,bytes)" \
  $ORACLE_DATA \
  $SIGNATURE \
  --private-key $PRIVATE_KEY \
  --rpc-url $RPC_URL
```

### Using a script

The `oracleData` is an ABI-encoded tuple of `(uint256 marketId, uint8 outcome, uint256 nonce, uint256 expiry)`.
The `signature` is a 65-byte `[r(32) || s(32) || v(1)]` ECDSA signature over the EIP-712 typed data hash.

Any EVM client (ethers.js, viem, go-ethereum) can submit these values directly.

---

## Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PRICING_SIGNER_PRIVATE_KEY` | Yes | — | Private key of the oracle signer (hex, with or without `0x` prefix) |
| `ORACLE_REGISTRY_ADDRESS` | Yes | — | Deployed OracleRegistry contract address |
| `CHAIN_ID` | Yes | — | EVM chain ID (falls back to `MONAD_TESTNET_CHAIN_ID`) |
| `PRICING_HTTP_ADDR` | No | `:8080` | HTTP listen address |
| `SIGNATURE_TTL_SECONDS` | No | `300` | Seconds until a signed payload expires |

---

## Nonce Storage: In-Memory vs Postgres

### Current State (MVP)

The pricing service uses `MemoryNonceStore` — a simple in-memory counter protected by a mutex.
This is fine for local development, but has a critical limitation:

> **Nonces reset on restart.** If the service restarts, previously issued nonces may be
> reissued, and the contract will reject them as replays (`NonceAlreadyUsed`).

### Postgres Nonce Store (Pre-Testnet)

Before any real testnet usage, switch to persistent nonce storage. The schema:

```sql
CREATE TABLE oracle_nonces (
    id          BIGSERIAL PRIMARY KEY,
    nonce       NUMERIC(78,0) NOT NULL UNIQUE,
    market_id   NUMERIC(78,0) NOT NULL,
    outcome     SMALLINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oracle_nonces_market ON oracle_nonces(market_id);
```

### Implementation Path

1. Create `PostgresNonceStore` implementing the existing `NonceStore` interface:

   ```go
   type NonceStore interface {
       Next(ctx context.Context) (*big.Int, error)
   }
   ```

2. Use `SELECT MAX(nonce) FROM oracle_nonces` + 1 with advisory locking to guarantee
   uniqueness across restarts and concurrent instances.

3. Insert each issued nonce with its market ID and outcome for audit trail.

4. The existing `docker-compose.yml` already provisions Postgres — no infrastructure changes needed.

### Migration Trigger

Switch from `MemoryNonceStore` to `PostgresNonceStore` **before deploying to any public testnet**.
The swap is a single-line change in `cmd/pricing/main.go`:

```diff
- pricing.NewMemoryNonceStore(nil),
+ storage.NewPostgresNonceStore(pool),
```

The `NonceStore` interface is designed for exactly this drop-in replacement.
