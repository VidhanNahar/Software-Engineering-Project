-- Upgrade existing stock schema for NSE bhavcopy integration.
-- Safe to run multiple times.

ALTER TABLE stock ADD COLUMN IF NOT EXISTS symbol VARCHAR(32);
ALTER TABLE stock ADD COLUMN IF NOT EXISTS series VARCHAR(8) NOT NULL DEFAULT 'EQ';
ALTER TABLE stock ADD COLUMN IF NOT EXISTS isin VARCHAR(32);
ALTER TABLE stock ADD COLUMN IF NOT EXISTS previous_close NUMERIC(15,2);
ALTER TABLE stock ADD COLUMN IF NOT EXISTS open_price NUMERIC(15,2);
ALTER TABLE stock ADD COLUMN IF NOT EXISTS day_high NUMERIC(15,2);
ALTER TABLE stock ADD COLUMN IF NOT EXISTS day_low NUMERIC(15,2);
ALTER TABLE stock ADD COLUMN IF NOT EXISTS close_price NUMERIC(15,2);
ALTER TABLE stock ADD COLUMN IF NOT EXISTS last_traded_price NUMERIC(15,2);
ALTER TABLE stock ADD COLUMN IF NOT EXISTS total_traded_qty BIGINT;
ALTER TABLE stock ADD COLUMN IF NOT EXISTS total_traded_value NUMERIC(20,2);
ALTER TABLE stock ADD COLUMN IF NOT EXISTS total_trades BIGINT;
ALTER TABLE stock ADD COLUMN IF NOT EXISTS trade_date DATE;

-- Fill symbol for legacy rows where only name existed.
UPDATE stock
SET symbol = UPPER(REPLACE(name, ' ', '_'))
WHERE symbol IS NULL OR symbol = '';

-- Enforce symbol requirement after backfill.
ALTER TABLE stock ALTER COLUMN symbol SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'stock_symbol_unique'
    ) THEN
        ALTER TABLE stock ADD CONSTRAINT stock_symbol_unique UNIQUE (symbol);
    END IF;
END$$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'stock'
          AND column_name = 'quantity'
          AND data_type <> 'bigint'
    ) THEN
        ALTER TABLE stock
        ALTER COLUMN quantity TYPE BIGINT USING quantity::BIGINT;
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'stock_day_range_valid'
    ) THEN
        ALTER TABLE stock
        ADD CONSTRAINT stock_day_range_valid
        CHECK (day_high IS NULL OR day_low IS NULL OR day_high >= day_low);
    END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_stock_symbol ON stock(symbol);
CREATE INDEX IF NOT EXISTS idx_stock_trade_date ON stock(trade_date DESC);

CREATE TABLE IF NOT EXISTS stock_daily_data (
    stock_daily_id       UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    stock_id             UUID            NOT NULL REFERENCES stock(stock_id) ON DELETE CASCADE,
    trade_date           DATE            NOT NULL,
    series               VARCHAR(8)      NOT NULL,
    open_price           NUMERIC(15,2)   CHECK (open_price >= 0),
    day_high             NUMERIC(15,2)   CHECK (day_high >= 0),
    day_low              NUMERIC(15,2)   CHECK (day_low >= 0),
    close_price          NUMERIC(15,2)   CHECK (close_price >= 0),
    last_traded_price    NUMERIC(15,2)   CHECK (last_traded_price >= 0),
    previous_close       NUMERIC(15,2)   CHECK (previous_close >= 0),
    total_traded_qty     BIGINT          CHECK (total_traded_qty >= 0),
    total_traded_value   NUMERIC(20,2)   CHECK (total_traded_value >= 0),
    total_trades         BIGINT          CHECK (total_trades >= 0),
    inserted_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT stock_daily_data_unique UNIQUE (stock_id, trade_date),
    CONSTRAINT stock_daily_day_range_valid CHECK (day_high IS NULL OR day_low IS NULL OR day_high >= day_low)
);

CREATE INDEX IF NOT EXISTS idx_stock_daily_trade_date ON stock_daily_data(trade_date DESC);
