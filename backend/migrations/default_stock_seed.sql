-- Seed a default stock universe for local/dev environments.
-- Safe to run multiple times and does not overwrite existing rows.

WITH seed (
    symbol,
    name,
    series,
    isin,
    price,
    previous_close,
    open_price,
    day_high,
    day_low,
    close_price,
    last_traded_price,
    total_traded_qty,
    total_traded_value,
    total_trades,
    quantity
) AS (
    VALUES
        ('RELIANCE', 'Reliance Industries Ltd', 'EQ', 'INE002A01018', 2948.35, 2920.20, 2928.00, 2964.60, 2912.10, 2942.80, 2948.35, 1850000, 5450000000, 112340, 1200000),
        ('TCS', 'Tata Consultancy Services Ltd', 'EQ', 'INE467B01029', 4212.75, 4194.20, 4201.00, 4230.00, 4186.15, 4208.10, 4212.75, 920000, 3870000000, 73400, 900000),
        ('INFY', 'Infosys Ltd', 'EQ', 'INE009A01021', 1865.40, 1848.00, 1852.25, 1873.60, 1845.10, 1862.30, 1865.40, 1100000, 2050000000, 81220, 1000000),
        ('HDFCBANK', 'HDFC Bank Ltd', 'EQ', 'INE040A01034', 1687.90, 1668.10, 1672.40, 1698.80, 1661.25, 1682.30, 1687.90, 1450000, 2430000000, 99880, 1300000),
        ('ICICIBANK', 'ICICI Bank Ltd', 'EQ', 'INE090A01021', 1215.55, 1201.40, 1206.00, 1221.90, 1197.25, 1211.30, 1215.55, 1700000, 2060000000, 108450, 1400000),
        ('SBIN', 'State Bank of India', 'EQ', 'INE062A01020', 786.20, 779.90, 781.50, 792.00, 777.25, 784.10, 786.20, 2500000, 1960000000, 120330, 2200000),
        ('HINDUNILVR', 'Hindustan Unilever Ltd', 'EQ', 'INE030A01027', 2532.10, 2518.45, 2521.00, 2544.80, 2512.00, 2528.90, 2532.10, 640000, 1620000000, 56210, 700000),
        ('ITC', 'ITC Ltd', 'EQ', 'INE154A01025', 438.65, 435.80, 436.20, 441.10, 434.60, 437.55, 438.65, 3200000, 1400000000, 136900, 3000000),
        ('LT', 'Larsen & Toubro Ltd', 'EQ', 'INE018A01030', 3738.95, 3712.50, 3720.20, 3755.00, 3698.40, 3731.80, 3738.95, 780000, 2910000000, 64820, 750000),
        ('KOTAKBANK', 'Kotak Mahindra Bank Ltd', 'EQ', 'INE237A01028', 1781.40, 1768.00, 1770.75, 1792.30, 1763.40, 1776.20, 1781.40, 850000, 1510000000, 59050, 820000),
        ('BHARTIARTL', 'Bharti Airtel Ltd', 'EQ', 'INE397D01024', 1328.25, 1311.80, 1316.20, 1335.40, 1309.70, 1324.60, 1328.25, 1550000, 2050000000, 90340, 1400000),
        ('WIPRO', 'Wipro Ltd', 'EQ', 'INE075A01022', 524.15, 518.90, 520.10, 526.80, 517.20, 523.00, 524.15, 1300000, 680000000, 71210, 1250000)
)
INSERT INTO stock (
    symbol,
    name,
    series,
    isin,
    price,
    previous_close,
    open_price,
    day_high,
    day_low,
    close_price,
    last_traded_price,
    total_traded_qty,
    total_traded_value,
    total_trades,
    trade_date,
    timestamp,
    quantity
)
SELECT
    symbol,
    name,
    series,
    isin,
    price,
    previous_close,
    open_price,
    day_high,
    day_low,
    close_price,
    last_traded_price,
    total_traded_qty,
    total_traded_value,
    total_trades,
    CURRENT_DATE,
    NOW(),
    quantity
FROM seed
ON CONFLICT (symbol) DO NOTHING;

INSERT INTO stock_daily_data (
    stock_id,
    trade_date,
    series,
    open_price,
    day_high,
    day_low,
    close_price,
    last_traded_price,
    previous_close,
    total_traded_qty,
    total_traded_value,
    total_trades
)
SELECT
    st.stock_id,
    COALESCE(st.trade_date, CURRENT_DATE),
    st.series,
    st.open_price,
    st.day_high,
    st.day_low,
    st.close_price,
    st.last_traded_price,
    st.previous_close,
    st.total_traded_qty,
    st.total_traded_value,
    st.total_trades
FROM stock st
WHERE st.symbol IN (
    'RELIANCE', 'TCS', 'INFY', 'HDFCBANK', 'ICICIBANK', 'SBIN',
    'HINDUNILVR', 'ITC', 'LT', 'KOTAKBANK', 'BHARTIARTL', 'WIPRO'
)
ON CONFLICT (stock_id, trade_date) DO NOTHING;
