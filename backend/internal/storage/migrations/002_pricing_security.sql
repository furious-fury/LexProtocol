CREATE SEQUENCE IF NOT EXISTS pricing_nonce_seq START WITH 1 INCREMENT BY 1;

CREATE TABLE IF NOT EXISTS pricing_nonces (
    nonce NUMERIC(78,0) PRIMARY KEY,
    market_id NUMERIC(78,0) NOT NULL,
    outcome SMALLINT NOT NULL CHECK (outcome IN (1, 2)),
    expiry BIGINT NOT NULL,
    signature TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pricing_nonces_market ON pricing_nonces(market_id);
