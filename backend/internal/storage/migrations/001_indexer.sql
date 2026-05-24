CREATE TABLE IF NOT EXISTS blocks (
    block_number BIGINT PRIMARY KEY,
    block_hash TEXT NOT NULL,
    parent_hash TEXT NOT NULL,
    confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pending_events (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    address TEXT NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash TEXT NOT NULL,
    tx_hash TEXT NOT NULL,
    log_index BIGINT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tx_hash, log_index)
);

CREATE TABLE IF NOT EXISTS confirmed_events (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    address TEXT NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash TEXT NOT NULL,
    tx_hash TEXT NOT NULL,
    log_index BIGINT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tx_hash, log_index)
);

CREATE TABLE IF NOT EXISTS markets (
    market_id NUMERIC(78,0) PRIMARY KEY,
    market_address TEXT NOT NULL,
    creator TEXT NOT NULL,
    status TEXT NOT NULL,
    lock_time BIGINT NOT NULL,
    resolution_rule TEXT NOT NULL,
    created_block_number BIGINT NOT NULL,
    created_tx_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS trades (
    id BIGSERIAL PRIMARY KEY,
    market_id NUMERIC(78,0) NOT NULL,
    user_address TEXT NOT NULL,
    side TEXT NOT NULL,
    amount NUMERIC(78,0) NOT NULL,
    tx_hash TEXT NOT NULL,
    log_index BIGINT NOT NULL,
    block_number BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tx_hash, log_index)
);

CREATE TABLE IF NOT EXISTS oracle_submissions (
    id BIGSERIAL PRIMARY KEY,
    market_id NUMERIC(78,0) NOT NULL,
    outcome TEXT NOT NULL,
    nonce NUMERIC(78,0) NOT NULL,
    expiry BIGINT NOT NULL,
    tx_hash TEXT NOT NULL,
    log_index BIGINT NOT NULL,
    block_number BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tx_hash, log_index)
);

CREATE TABLE IF NOT EXISTS settlements (
    id BIGSERIAL PRIMARY KEY,
    market_id NUMERIC(78,0) NOT NULL,
    outcome TEXT NOT NULL,
    tx_hash TEXT NOT NULL,
    log_index BIGINT NOT NULL,
    block_number BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tx_hash, log_index)
);

CREATE TABLE IF NOT EXISTS redemptions (
    id BIGSERIAL PRIMARY KEY,
    market_id NUMERIC(78,0) NOT NULL,
    user_address TEXT NOT NULL,
    amount NUMERIC(78,0) NOT NULL,
    tx_hash TEXT NOT NULL,
    log_index BIGINT NOT NULL,
    block_number BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_pending_events_block ON pending_events(block_number);
CREATE INDEX IF NOT EXISTS idx_confirmed_events_block ON confirmed_events(block_number);
CREATE INDEX IF NOT EXISTS idx_trades_market ON trades(market_id);
CREATE INDEX IF NOT EXISTS idx_oracle_submissions_market ON oracle_submissions(market_id);
CREATE INDEX IF NOT EXISTS idx_settlements_market ON settlements(market_id);
CREATE INDEX IF NOT EXISTS idx_redemptions_market ON redemptions(market_id);
