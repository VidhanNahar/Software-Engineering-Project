import io
import zipfile
from datetime import datetime, timezone
from decimal import Decimal
import os

import pandas as pd
import psycopg2
import requests
from psycopg2.extras import execute_values

START_DATE = datetime(2025, 4, 1)
END_DATE = datetime.today()
BASE_ARCHIVE_URL = "https://nsearchives.nseindia.com/content/cm"
NSE_HOME = "https://www.nseindia.com/"
VALID_SERIES = {"EQ", "BE", "BZ", "SM", "ST"}


def db_connect():
    return psycopg2.connect(
        host=os.getenv("DB_HOST", "localhost"),
        port=os.getenv("DB_PORT", "5432"),
        user=os.getenv("DB_USER", "myuser"),
        password=os.getenv("DB_PASSWORD", "mypassword"),
        dbname=os.getenv("DB_NAME", "project_db"),
    )


def first_existing_col(df: pd.DataFrame, candidates: list[str]) -> str | None:
    cols = {str(c).strip().upper(): c for c in df.columns}
    for c in candidates:
        if c in cols:
            return cols[c]
    return None


def to_float(value) -> float:
    if value is None:
        return 0.0
    s = str(value).strip().replace(",", "")
    if s in {"", "-", "nan", "NaN", "None"}:
        return 0.0
    try:
        return float(s)
    except Exception:
        return 0.0


def to_int(value) -> int:
    if value is None:
        return 0
    s = str(value).strip().replace(",", "")
    if s in {"", "-", "nan", "NaN", "None"}:
        return 0
    try:
        return int(float(s))
    except Exception:
        return 0


def parse_trade_date(raw: str, fallback_dt: pd.Timestamp) -> datetime:
    s = str(raw).strip()
    for fmt in ("%d-%b-%Y", "%d-%B-%Y", "%Y-%m-%d", "%d/%m/%Y", "%Y%m%d"):
        try:
            return datetime.strptime(s, fmt)
        except Exception:
            pass
    return fallback_dt.to_pydatetime()


def download_day_df(session: requests.Session, dt: pd.Timestamp) -> pd.DataFrame | None:
    ymd = dt.strftime("%Y%m%d")
    ddmmyy = dt.strftime("%d%m%y")
    urls = [
        f"{BASE_ARCHIVE_URL}/BhavCopy_NSE_CM_0_0_0_{ymd}_F_0000.csv.zip",
        f"{BASE_ARCHIVE_URL}/cm{ddmmyy}bhav.csv.zip",
    ]

    response = None
    for url in urls:
        r = session.get(url, timeout=30)
        if r.status_code == 200:
            response = r
            break

    if response is None:
        return None

    with zipfile.ZipFile(io.BytesIO(response.content)) as zf:
        csv_names = [n for n in zf.namelist() if n.lower().endswith(".csv")]
        if not csv_names:
            return None
        with zf.open(csv_names[0]) as csv_file:
            return pd.read_csv(csv_file)


def normalize_day(df: pd.DataFrame, dt: pd.Timestamp) -> list[dict]:
    symbol_col = first_existing_col(df, ["SYMBOL", "TCKRSYMB"])
    series_col = first_existing_col(df, ["SERIES", "SCTYSRS"])
    name_col = first_existing_col(df, ["NAME OF COMPANY", "SECURITY", "FININSTRMNM"])
    isin_col = first_existing_col(df, ["ISIN"])
    open_col = first_existing_col(df, ["OPEN", "OPNPRIC"])
    high_col = first_existing_col(df, ["HIGH", "HGHPRIC"])
    low_col = first_existing_col(df, ["LOW", "LWPRIC"])
    close_col = first_existing_col(df, ["CLOSE", "CLSPRIC"])
    last_col = first_existing_col(df, ["LAST", "LASTPRIC"])
    prev_col = first_existing_col(df, ["PREVCLOSE", "PRVSCLSGPRIC"])
    qty_col = first_existing_col(df, ["TOTTRDQTY", "TTLTRADGVOL"])
    val_col = first_existing_col(df, ["TOTTRDVAL", "TTLTRFVAL"])
    trades_col = first_existing_col(df, ["TOTALTRADES", "TTLNBOFTXSEXCTD"])
    trade_date_col = first_existing_col(df, ["TIMESTAMP", "TRADDT", "BIZDT"])

    if symbol_col is None or series_col is None:
        return []

    out = []
    for _, row in df.iterrows():
        symbol = str(row.get(symbol_col, "")).strip().upper()
        series = str(row.get(series_col, "")).strip().upper()
        if not symbol or series not in VALID_SERIES:
            continue

        name_raw = str(row.get(name_col, symbol)).strip() if name_col else symbol
        name = (name_raw if name_raw else symbol)[:128]
        isin = str(row.get(isin_col, "")).strip() if isin_col else ""

        open_price = to_float(row.get(open_col)) if open_col else 0.0
        high = to_float(row.get(high_col)) if high_col else 0.0
        low = to_float(row.get(low_col)) if low_col else 0.0
        close = to_float(row.get(close_col)) if close_col else 0.0
        last = to_float(row.get(last_col)) if last_col else 0.0
        prev_close = to_float(row.get(prev_col)) if prev_col else 0.0

        if last <= 0:
            last = close
        if last <= 0:
            continue

        qty = to_int(row.get(qty_col)) if qty_col else 0
        traded_value = to_float(row.get(val_col)) if val_col else 0.0
        total_trades = to_int(row.get(trades_col)) if trades_col else 0

        raw_date = str(row.get(trade_date_col, "")).strip() if trade_date_col else ""
        trade_date = parse_trade_date(raw_date, dt)

        out.append(
            {
                "symbol": symbol,
                "name": name,
                "series": series,
                "isin": isin,
                "open": open_price,
                "high": high,
                "low": low,
                "close": close,
                "last": last,
                "prev_close": prev_close,
                "qty": qty,
                "traded_value": traded_value,
                "total_trades": total_trades,
                "trade_date": trade_date.date(),
            }
        )

    return out


def main() -> None:
    session = requests.Session()
    session.headers.update(
        {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
            "Accept-Language": "en-US,en;q=0.9",
            "Referer": NSE_HOME,
            "Connection": "keep-alive",
        }
    )

    try:
        session.get(NSE_HOME, timeout=20)
    except Exception:
        pass

    business_days = pd.bdate_range(start=START_DATE, end=END_DATE)

    conn = db_connect()
    conn.autocommit = False

    upsert_sql = """
    INSERT INTO stock (
        symbol, name, series, isin, price, previous_close, open_price, day_high, day_low,
        close_price, last_traded_price, total_traded_qty, total_traded_value, total_trades,
        trade_date, timestamp, quantity
    ) VALUES %s
    ON CONFLICT (symbol)
    DO UPDATE SET
        name = EXCLUDED.name,
        series = EXCLUDED.series,
        isin = COALESCE(NULLIF(EXCLUDED.isin, ''), stock.isin),
        price = EXCLUDED.price,
        previous_close = EXCLUDED.previous_close,
        open_price = EXCLUDED.open_price,
        day_high = EXCLUDED.day_high,
        day_low = EXCLUDED.day_low,
        close_price = EXCLUDED.close_price,
        last_traded_price = EXCLUDED.last_traded_price,
        total_traded_qty = EXCLUDED.total_traded_qty,
        total_traded_value = EXCLUDED.total_traded_value,
        total_trades = EXCLUDED.total_trades,
        trade_date = EXCLUDED.trade_date,
        timestamp = NOW(),
        quantity = GREATEST(COALESCE(stock.quantity, 0), EXCLUDED.quantity)
    """

    insert_daily_sql = """
    INSERT INTO stock_daily_data (
        stock_id, trade_date, series, open_price, day_high, day_low,
        close_price, last_traded_price, previous_close, total_traded_qty,
        total_traded_value, total_trades
    )
    SELECT
        s.stock_id, d.trade_date, d.series, d.open_price, d.day_high, d.day_low,
        d.close_price, d.last_traded_price, d.previous_close, d.total_traded_qty,
        d.total_traded_value, d.total_trades
    FROM (VALUES %s) AS d(
        symbol, trade_date, series, open_price, day_high, day_low,
        close_price, last_traded_price, previous_close, total_traded_qty,
        total_traded_value, total_trades
    )
    JOIN stock s ON s.symbol = d.symbol
    ON CONFLICT (stock_id, trade_date)
    DO UPDATE SET
        series = EXCLUDED.series,
        open_price = EXCLUDED.open_price,
        day_high = EXCLUDED.day_high,
        day_low = EXCLUDED.day_low,
        close_price = EXCLUDED.close_price,
        last_traded_price = EXCLUDED.last_traded_price,
        previous_close = EXCLUDED.previous_close,
        total_traded_qty = EXCLUDED.total_traded_qty,
        total_traded_value = EXCLUDED.total_traded_value,
        total_trades = EXCLUDED.total_trades
    """

    success_days = 0
    closed_holidays = 0
    parser_errors = 0
    total_rows = 0

    with conn.cursor() as cur:
        for dt in business_days:
            try:
                day_df = download_day_df(session, dt)
                if day_df is None:
                    closed_holidays += 1
                    continue

                normalized = normalize_day(day_df, dt)
                if not normalized:
                    parser_errors += 1
                    continue

                stock_values = [
                    (
                        r["symbol"],
                        r["name"],
                        r["series"],
                        r["isin"],
                        Decimal(str(r["last"])),
                        Decimal(str(r["prev_close"])),
                        Decimal(str(r["open"])),
                        Decimal(str(r["high"])),
                        Decimal(str(r["low"])),
                        Decimal(str(r["close"])),
                        Decimal(str(r["last"])),
                        r["qty"],
                        Decimal(str(r["traded_value"])),
                        r["total_trades"],
                        r["trade_date"],
                        datetime.now(timezone.utc),
                        r["qty"],
                    )
                    for r in normalized
                ]
                execute_values(cur, upsert_sql, stock_values, page_size=1000)

                daily_values = [
                    (
                        r["symbol"],
                        r["trade_date"],
                        r["series"],
                        Decimal(str(r["open"])),
                        Decimal(str(r["high"])),
                        Decimal(str(r["low"])),
                        Decimal(str(r["close"])),
                        Decimal(str(r["last"])),
                        Decimal(str(r["prev_close"])),
                        r["qty"],
                        Decimal(str(r["traded_value"])),
                        r["total_trades"],
                    )
                    for r in normalized
                ]
                execute_values(cur, insert_daily_sql, daily_values, page_size=1000)

                conn.commit()
                success_days += 1
                total_rows += len(normalized)
            except Exception as e:
                conn.rollback()
                parser_errors += 1
                print(f"IMPORT_ERROR_DAY={dt.strftime('%Y-%m-%d')}")
                print(f"IMPORT_ERROR={e}")

    conn.close()

    print(f"TRADING_DAYS_CHECKED={len(business_days)}")
    print(f"SUCCESS_DAYS={success_days}")
    print(f"CLOSED_HOLIDAY_DAYS={closed_holidays}")
    print(f"PARSER_ERROR_DAYS={parser_errors}")
    print(f"ROWS_IMPORTED={total_rows}")


if __name__ == "__main__":
    main()
