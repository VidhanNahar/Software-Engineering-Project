-- =============================================================
-- FinXGrow Tick + Candle Time-Series Tables
-- =============================================================

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS stock_ticks (
    tick_id        BIGSERIAL       PRIMARY KEY,
    stock_id       UUID            NOT NULL REFERENCES stock(stock_id) ON DELETE CASCADE,
    symbol         VARCHAR(32)     NOT NULL,
    tick_time      TIMESTAMPTZ     NOT NULL,
    price          NUMERIC(15,4)   NOT NULL CHECK (price > 0),
    volume         BIGINT          NOT NULL DEFAULT 0 CHECK (volume >= 0),
    trade_value    NUMERIC(20,4)   NOT NULL DEFAULT 0 CHECK (trade_value >= 0),
    market_open    BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    PERFORM create_hypertable('stock_ticks', 'tick_time', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN NULL;
    WHEN insufficient_privilege THEN NULL;
    WHEN OTHERS THEN NULL;
END$$;

CREATE INDEX IF NOT EXISTS idx_stock_ticks_symbol_time ON stock_ticks(symbol, tick_time DESC);
CREATE INDEX IF NOT EXISTS idx_stock_ticks_stock_time ON stock_ticks(stock_id, tick_time DESC);

CREATE TABLE IF NOT EXISTS stock_candles (
    candle_id       BIGSERIAL       PRIMARY KEY,
    stock_id        UUID            NOT NULL REFERENCES stock(stock_id) ON DELETE CASCADE,
    symbol          VARCHAR(32)     NOT NULL,
    timeframe       VARCHAR(8)      NOT NULL,
    candle_time     TIMESTAMPTZ     NOT NULL,
    open            NUMERIC(15,4)   NOT NULL CHECK (open > 0),
    high            NUMERIC(15,4)   NOT NULL CHECK (high > 0),
    low             NUMERIC(15,4)   NOT NULL CHECK (low > 0),
    close           NUMERIC(15,4)   NOT NULL CHECK (close > 0),
    volume          BIGINT          NOT NULL DEFAULT 0 CHECK (volume >= 0),
    tick_count      BIGINT          NOT NULL DEFAULT 0 CHECK (tick_count >= 0),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT stock_candles_unique UNIQUE (symbol, timeframe, candle_time)
);

DO $$
BEGIN
    PERFORM create_hypertable('stock_candles', 'candle_time', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN NULL;
    WHEN insufficient_privilege THEN NULL;
    WHEN OTHERS THEN NULL;
END$$;

CREATE INDEX IF NOT EXISTS idx_stock_candles_symbol_tf_time ON stock_candles(symbol, timeframe, candle_time DESC);
CREATE INDEX IF NOT EXISTS idx_stock_candles_stock_tf_time ON stock_candles(stock_id, timeframe, candle_time DESC);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM timescaledb_information.hypertables
        WHERE hypertable_name = 'stock_ticks'
    ) THEN
        ALTER TABLE stock_ticks SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'symbol'
        );
    END IF;
EXCEPTION
    WHEN undefined_table THEN NULL;
    WHEN insufficient_privilege THEN NULL;
    WHEN feature_not_supported THEN NULL;
    WHEN OTHERS THEN NULL;
END$$;
DO $$
BEGIN
    PERFORM add_compression_policy('stock_ticks', INTERVAL '7 days', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN NULL;
    WHEN insufficient_privilege THEN NULL;
END$$;

DO $$
BEGIN
    PERFORM add_retention_policy('stock_ticks', INTERVAL '90 days', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN NULL;
    WHEN insufficient_privilege THEN NULL;
END$$;
