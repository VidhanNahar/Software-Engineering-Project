import io
import zipfile
from datetime import datetime

import pandas as pd
import requests

START_DATE = datetime(2025, 4, 1)
END_DATE = datetime.today()
BASE_ARCHIVE_URL = "https://nsearchives.nseindia.com/content/cm"
NSE_HOME = "https://www.nseindia.com/"
VALID_SERIES = {"EQ", "BE", "BZ", "SM", "ST"}


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

    # Business-day range skips Saturdays and Sundays.
    date_list = pd.bdate_range(start=START_DATE, end=END_DATE)
    unique_symbols: set[str] = set()
    success_days = 0
    closed_holiday_days = 0
    parser_error_days = 0

    for dt in date_list:
        ymd = dt.strftime("%Y%m%d")
        ddmmyy = dt.strftime("%d%m%y")
        urls = [
            f"{BASE_ARCHIVE_URL}/BhavCopy_NSE_CM_0_0_0_{ymd}_F_0000.csv.zip",
            f"{BASE_ARCHIVE_URL}/cm{ddmmyy}bhav.csv.zip",
        ]

        try:
            response = None
            for url in urls:
                r = session.get(url, timeout=30)
                if r.status_code == 200:
                    response = r
                    break

            if response is None:
                # Weekday with no bhavcopy is treated as exchange-closed holiday.
                closed_holiday_days += 1
                continue

            with zipfile.ZipFile(io.BytesIO(response.content)) as zf:
                csv_names = [n for n in zf.namelist() if n.lower().endswith(".csv")]
                if not csv_names:
                    parser_error_days += 1
                    continue

                with zf.open(csv_names[0]) as csv_file:
                    df = pd.read_csv(csv_file)

            cols = {str(c).strip().upper(): c for c in df.columns}

            if "SERIES" in cols:
                df = df[df[cols["SERIES"]].astype(str).str.upper().isin(VALID_SERIES)]
            elif "SCTYSRS" in cols:
                df = df[df[cols["SCTYSRS"]].astype(str).str.upper().isin(VALID_SERIES)]

            symbol_col = None
            if "SYMBOL" in cols:
                symbol_col = cols["SYMBOL"]
            elif "TCKRSYMB" in cols:
                symbol_col = cols["TCKRSYMB"]

            if symbol_col is not None:
                symbols = (
                    df[symbol_col]
                    .dropna()
                    .astype(str)
                    .str.strip()
                    .str.upper()
                )
                unique_symbols.update(sym for sym in symbols if sym)

            success_days += 1
        except Exception:
            parser_error_days += 1

    print(f"UNIQUE_STOCKS={len(unique_symbols)}")
    print(f"TRADING_DAYS_CHECKED={len(date_list)}")
    print(f"SUCCESS_DAYS={success_days}")
    print(f"CLOSED_HOLIDAY_DAYS={closed_holiday_days}")
    print(f"PARSER_ERROR_DAYS={parser_error_days}")


if __name__ == "__main__":
    main()
