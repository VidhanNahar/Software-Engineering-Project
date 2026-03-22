-- =============================================================
-- Market Status Table
-- =============================================================
-- Tracks if the market is open or closed for trading
-- When market is closed, no buy/sell transactions are allowed

CREATE TABLE IF NOT EXISTS market_status (
    market_id         UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    is_open           BOOLEAN         NOT NULL DEFAULT FALSE,
    opened_at         TIMESTAMPTZ,
    closed_at         TIMESTAMPTZ,
    total_trades      BIGINT          NOT NULL DEFAULT 0,
    total_volume      BIGINT          NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- Create index for quick lookups
CREATE INDEX IF NOT EXISTS idx_market_status_is_open ON market_status(is_open);

-- Insert default market status (closed)
INSERT INTO market_status (is_open) 
SELECT FALSE 
WHERE NOT EXISTS (SELECT 1 FROM market_status);
