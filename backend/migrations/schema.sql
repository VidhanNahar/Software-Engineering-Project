-- =============================================================
-- FinXGrow - Full Database Schema
-- Derived from design_doc.pdf
-- =============================================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Custom ENUM for order status
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_status') THEN
        CREATE TYPE order_status AS ENUM ('Pending', 'Filed', 'Cancelled');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('guest', 'user', 'admin');
    END IF;
END$$;

-- =============================================================
-- 1. Users
-- =============================================================
CREATE TABLE IF NOT EXISTS users (
    user_id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name              VARCHAR(255)  NOT NULL,
    email_id          VARCHAR(255)  UNIQUE,
    password          VARCHAR(255)  NOT NULL,
    role              user_role     NOT NULL DEFAULT 'guest',
    aadhar_id         NUMERIC(12,0) UNIQUE,
    pan_id            VARCHAR(10)   UNIQUE,
    phone_number      NUMERIC(10,0) UNIQUE,
    date_of_birth     DATE          NOT NULL,
    is_verified_email BOOLEAN       NOT NULL DEFAULT FALSE,
    is_kyc_verified   BOOLEAN       NOT NULL DEFAULT FALSE
);

-- =============================================================
-- 2. Stock
-- =============================================================
CREATE TABLE IF NOT EXISTS stock (
    stock_id            UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(128)    NOT NULL,
    price               NUMERIC(15,2)   CHECK (price > 0),
    timestamp           TIMESTAMPTZ     NOT NULL,
    quantity            BIGINT
);

-- =============================================================
-- 3. Portfolio
-- =============================================================
CREATE TABLE IF NOT EXISTS portfolio (
    portfolio_id      UUID           NOT NULL DEFAULT gen_random_uuid(),
    portfolio_name    VARCHAR(255),
    user_id           UUID           REFERENCES users(user_id),
    stock_id          UUID           REFERENCES stock(stock_id),
    transaction_time  TIMESTAMPTZ    NOT NULL,
    price             NUMERIC(15,2)  CHECK (price > 0),
    quantity          INTEGER        CHECK (quantity > 0),

    PRIMARY KEY (portfolio_id, stock_id, transaction_time)
);

-- =============================================================
-- 4. WatchList
-- =============================================================
CREATE TABLE IF NOT EXISTS watchlist (
    watchlist_id   UUID           NOT NULL DEFAULT gen_random_uuid(),
    user_id        UUID           REFERENCES users(user_id),
    watchlist_name VARCHAR(255)   NOT NULL,
    stock_id       UUID           REFERENCES stock(stock_id),
    quantity       INTEGER        CHECK (quantity > 0),
    price          NUMERIC(15,2)  CHECK (price > 0),
    timestamp      TIMESTAMPTZ    NOT NULL,

    PRIMARY KEY (watchlist_id, stock_id)
);

-- =============================================================
-- 5. Orders
-- =============================================================
CREATE TABLE IF NOT EXISTS orders (
    order_id        UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    stock_id        UUID           REFERENCES stock(stock_id),
    user_id         UUID           REFERENCES users(user_id),
    timestamp       TIMESTAMPTZ    NOT NULL,
    status          order_status   NOT NULL,
    quantity        INTEGER        CHECK (quantity > 0),
    price_per_stock NUMERIC(15,2)  CHECK (price_per_stock > 0)
);

-- =============================================================
-- 6. Wallet
-- =============================================================
CREATE TABLE IF NOT EXISTS wallet (
    wallet_id      UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID           REFERENCES users(user_id),
    balance        NUMERIC(15,2)  CHECK (balance >= 0),
    locked_balance NUMERIC(15,2)  CHECK (locked_balance >= 0)
);

-- =============================================================
-- Transaction Safety Constraints
-- =============================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'wallet_user_id_unique'
    ) THEN
        ALTER TABLE wallet
        ADD CONSTRAINT wallet_user_id_unique UNIQUE (user_id);
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'portfolio_user_stock_unique'
    ) THEN
        ALTER TABLE portfolio
        ADD CONSTRAINT portfolio_user_stock_unique UNIQUE (user_id, stock_id);
    END IF;
END$$;
