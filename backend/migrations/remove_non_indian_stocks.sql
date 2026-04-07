-- Remove all non-Indian stocks from database
-- This migration ensures the system only handles INR-denominated Indian stocks from NSE

-- Step 1: Remove dependent records first (to avoid foreign key violations)

DELETE FROM stock_ticks WHERE symbol IN (SELECT symbol FROM stock WHERE currency_code <> 'INR');
DELETE FROM stock_candles WHERE symbol IN (SELECT symbol FROM stock WHERE currency_code <> 'INR');
DELETE FROM stock_daily_data WHERE stock_id IN (SELECT stock_id FROM stock WHERE currency_code <> 'INR');
DELETE FROM orders WHERE stock_id IN (SELECT stock_id FROM stock WHERE currency_code <> 'INR');
DELETE FROM portfolio WHERE stock_id IN (SELECT stock_id FROM stock WHERE currency_code <> 'INR');
DELETE FROM watchlist WHERE stock_id IN (SELECT stock_id FROM stock WHERE currency_code <> 'INR');
DELETE FROM alerts WHERE stock_id IN (SELECT stock_id FROM stock WHERE currency_code <> 'INR');

-- Step 2: Delete the non-Indian stocks themselves
DELETE FROM stock WHERE currency_code <> 'INR';

-- Step 3: Add constraint to prevent non-Indian stocks in the future
ALTER TABLE stock ADD CONSTRAINT stock_inr_only_check CHECK (currency_code = 'INR');
