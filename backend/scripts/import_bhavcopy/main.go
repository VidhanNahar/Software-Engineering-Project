package main

import (
	"archive/zip"
	"backend-go/database"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type bhavcopyRow struct {
	Symbol      string
	Name        string
	Series      string
	ISIN        string
	Open        float64
	High        float64
	Low         float64
	Close       float64
	Last        float64
	PrevClose   float64
	TradedQty   int64
	TradedValue float64
	TotalTrades int64
	TradeDate   time.Time
}

func main() {
	_ = godotenv.Load(".env")

	csvPath := flag.String("csv", "", "Path to bhavcopy CSV file (optional)")
	dateStr := flag.String("date", time.Now().Format("20060102"), "Bhavcopy date in YYYYMMDD")
	includeAllSeries := flag.Bool("all-series", true, "Import all series instead of only EQ")
	flag.Parse()

	db, err := database.Connect(
		envOrDefault("DB_HOST", "localhost"),
		envOrDefault("DB_PORT", "5432"),
		envOrDefault("DB_USER", "myuser"),
		envOrDefault("DB_PASSWORD", "mypassword"),
		envOrDefault("DB_NAME", "project_db"),
	)
	if err != nil {
		log.Fatalf("database connect failed: %v", err)
	}
	defer database.Close(db)

	var csvBytes []byte
	if strings.TrimSpace(*csvPath) != "" {
		csvBytes, err = os.ReadFile(*csvPath)
		if err != nil {
			log.Fatalf("failed to read csv file: %v", err)
		}
	} else {
		csvBytes, err = downloadBhavcopyCSV(*dateStr)
		if err != nil {
			log.Fatalf("failed to download bhavcopy: %v", err)
		}
	}

	rows, err := parseBhavcopy(csvBytes, *dateStr, *includeAllSeries)
	if err != nil {
		log.Fatalf("failed to parse bhavcopy: %v", err)
	}
	if len(rows) == 0 {
		log.Fatalf("no rows parsed from bhavcopy")
	}

	insertedOrUpdated, dailyInserted, err := importRows(db, rows)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	fmt.Printf("Imported rows: %d\n", insertedOrUpdated)
	fmt.Printf("Daily rows inserted: %d\n", dailyInserted)
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func downloadBhavcopyCSV(dateYYYYMMDD string) ([]byte, error) {
	urls := []string{
		fmt.Sprintf("https://nsearchives.nseindia.com/content/cm/BhavCopy_NSE_CM_0_0_0_%s_F_0000.csv.zip", dateYYYYMMDD),
		fmt.Sprintf("https://nsearchives.nseindia.com/content/historical/EQUITIES/%s/%s/cm%sbhav.csv.zip", strings.ToUpper(timeMonth(dateYYYYMMDD)), timeYear(dateYYYYMMDD), timeDay(dateYYYYMMDD)),
	}

	client := &http.Client{Timeout: 45 * time.Second}
	for _, u := range urls {
		b, err := fetchURL(client, u)
		if err != nil {
			continue
		}
		csvBytes, err := unzipFirstCSV(b)
		if err == nil {
			return csvBytes, nil
		}
	}

	return nil, errors.New("unable to download bhavcopy for date; provide -csv path")
}

func fetchURL(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0 Safari/537.36")
	req.Header.Set("Accept", "application/zip,application/octet-stream,*/*")
	req.Header.Set("Referer", "https://www.nseindia.com/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func unzipFirstCSV(zipBytes []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, err
	}

	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}

	return nil, errors.New("no csv file found in zip")
}

func parseBhavcopy(csvBytes []byte, fallbackDate string, includeAllSeries bool) ([]bhavcopyRow, error) {
	r := csv.NewReader(bytes.NewReader(csvBytes))
	r.FieldsPerRecord = -1

	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, errors.New("csv has no data rows")
	}

	headers := make(map[string]int)
	for i, h := range records[0] {
		headers[strings.ToUpper(strings.TrimSpace(h))] = i
	}

	must := []string{"SYMBOL", "SERIES", "OPEN", "HIGH", "LOW", "CLOSE"}
	for _, key := range must {
		if _, ok := headers[key]; !ok {
			return nil, fmt.Errorf("missing required column: %s", key)
		}
	}

	rows := make([]bhavcopyRow, 0, len(records)-1)
	for _, rec := range records[1:] {
		if len(rec) == 0 {
			continue
		}

		symbol := getCol(rec, headers, "SYMBOL")
		series := getCol(rec, headers, "SERIES")
		if symbol == "" {
			continue
		}
		if !includeAllSeries && strings.ToUpper(series) != "EQ" {
			continue
		}

		open := parseFloatSafe(getCol(rec, headers, "OPEN"))
		high := parseFloatSafe(getCol(rec, headers, "HIGH"))
		low := parseFloatSafe(getCol(rec, headers, "LOW"))
		closeP := parseFloatSafe(getCol(rec, headers, "CLOSE"))
		last := parseFloatSafe(getCol(rec, headers, "LAST"))
		if last == 0 {
			last = closeP
		}
		prevClose := parseFloatSafe(getCol(rec, headers, "PREVCLOSE"))
		qty := parseIntSafe(getCol(rec, headers, "TOTTRDQTY"))
		val := parseFloatSafe(getCol(rec, headers, "TOTTRDVAL"))
		trades := parseIntSafe(getCol(rec, headers, "TOTALTRADES"))
		isin := getCol(rec, headers, "ISIN")

		tradeDate := parseTradeDate(getCol(rec, headers, "TIMESTAMP"), fallbackDate)
		if tradeDate.IsZero() {
			tradeDate = time.Now().UTC()
		}

		name := symbol
		if sec := getCol(rec, headers, "SECURITY"); sec != "" {
			name = sec
		}

		rows = append(rows, bhavcopyRow{
			Symbol:      strings.ToUpper(symbol),
			Name:        name,
			Series:      strings.ToUpper(series),
			ISIN:        isin,
			Open:        open,
			High:        high,
			Low:         low,
			Close:       closeP,
			Last:        last,
			PrevClose:   prevClose,
			TradedQty:   qty,
			TradedValue: val,
			TotalTrades: trades,
			TradeDate:   tradeDate,
		})
	}

	return rows, nil
}

func getCol(rec []string, idx map[string]int, key string) string {
	i, ok := idx[key]
	if !ok || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

func parseFloatSafe(s string) float64 {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" || s == "-" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseIntSafe(s string) int64 {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" || s == "-" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		f, ferr := strconv.ParseFloat(s, 64)
		if ferr != nil {
			return 0
		}
		return int64(f)
	}
	return v
}

func parseTradeDate(raw, fallbackYYYYMMDD string) time.Time {
	raw = strings.TrimSpace(raw)
	layouts := []string{"02-JAN-2006", "02-Jan-2006", "2006-01-02", "02/01/2006", "20060102"}
	for _, l := range layouts {
		if t, err := time.Parse(l, strings.ToUpper(raw)); err == nil {
			return t.UTC()
		}
		if t, err := time.Parse(l, raw); err == nil {
			return t.UTC()
		}
	}
	if t, err := time.Parse("20060102", fallbackYYYYMMDD); err == nil {
		return t.UTC()
	}
	return time.Time{}
}

func importRows(db *sql.DB, rows []bhavcopyRow) (int, int, error) {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	upsertStockSQL := `
INSERT INTO stock (
    symbol, name, series, isin, price, previous_close, open_price, day_high, day_low,
    close_price, last_traded_price, total_traded_qty, total_traded_value, total_trades,
    trade_date, timestamp, quantity
)
VALUES (
    UPPER($1), $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14, $15, NOW(), $16
)
ON CONFLICT (symbol)
DO UPDATE SET
    name = EXCLUDED.name,
    series = EXCLUDED.series,
    isin = COALESCE(EXCLUDED.isin, stock.isin),
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
RETURNING stock_id;`

	insertDailySQL := `
INSERT INTO stock_daily_data (
    stock_id, trade_date, series, open_price, day_high, day_low,
    close_price, last_traded_price, previous_close, total_traded_qty,
    total_traded_value, total_trades
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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
    total_trades = EXCLUDED.total_trades;`

	upsertStmt, err := tx.PrepareContext(ctx, upsertStockSQL)
	if err != nil {
		return 0, 0, err
	}
	defer upsertStmt.Close()

	dailyStmt, err := tx.PrepareContext(ctx, insertDailySQL)
	if err != nil {
		return 0, 0, err
	}
	defer dailyStmt.Close()

	processed := 0
	dailyProcessed := 0
	for _, row := range rows {
		price := row.Last
		if price <= 0 {
			price = row.Close
		}
		if price <= 0 {
			continue
		}
		qty := row.TradedQty
		if qty < 0 {
			qty = 0
		}

		var stockID string
		err = upsertStmt.QueryRowContext(
			ctx,
			row.Symbol,
			row.Name,
			row.Series,
			row.ISIN,
			price,
			row.PrevClose,
			row.Open,
			row.High,
			row.Low,
			row.Close,
			row.Last,
			row.TradedQty,
			row.TradedValue,
			row.TotalTrades,
			row.TradeDate,
			qty,
		).Scan(&stockID)
		if err != nil {
			return 0, 0, err
		}

		_, err = dailyStmt.ExecContext(
			ctx,
			stockID,
			row.TradeDate,
			row.Series,
			row.Open,
			row.High,
			row.Low,
			row.Close,
			row.Last,
			row.PrevClose,
			row.TradedQty,
			row.TradedValue,
			row.TotalTrades,
		)
		if err != nil {
			return 0, 0, err
		}

		processed++
		dailyProcessed++
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}

	return processed, dailyProcessed, nil
}

func timeYear(dateYYYYMMDD string) string {
	if len(dateYYYYMMDD) >= 4 {
		return dateYYYYMMDD[0:4]
	}
	return time.Now().Format("2006")
}

func timeMonth(dateYYYYMMDD string) string {
	if len(dateYYYYMMDD) >= 6 {
		m, err := time.Parse("01", dateYYYYMMDD[4:6])
		if err == nil {
			return m.Format("Jan")
		}
	}
	return strings.ToUpper(time.Now().Format("Jan"))
}

func timeDay(dateYYYYMMDD string) string {
	if len(dateYYYYMMDD) >= 8 {
		return dateYYYYMMDD[6:8]
	}
	return time.Now().Format("02")
}

func init() {
	// Ensure script can find root .env when run from backend/scripts.
	if cwd, err := os.Getwd(); err == nil {
		if filepath.Base(cwd) == "scripts" {
			_ = os.Chdir("..")
		}
	}
}
