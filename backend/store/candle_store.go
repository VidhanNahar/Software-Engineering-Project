package store

import (
	"backend-go/model"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

func (s *Store) GetStockTicksBySymbol(symbol string, limit int) ([]model.StockTick, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(`
		SELECT symbol, tick_time, price, volume, trade_value
		FROM stock_ticks
		WHERE symbol = UPPER($1)
		ORDER BY tick_time DESC
		LIMIT $2`, symbol, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ticks := make([]model.StockTick, 0, limit)
	for rows.Next() {
		var t model.StockTick
		if err := rows.Scan(&t.Symbol, &t.TickTime, &t.Price, &t.Volume, &t.TradeValue); err != nil {
			return nil, err
		}
		ticks = append(ticks, t)
	}
	return ticks, rows.Err()
}

func (s *Store) GetStockCandlesBySymbol(symbol, timeframe string, limit int) ([]model.StockCandle, error) {
	if limit <= 0 {
		limit = 200
	}
	interval, err := normalizeTimeframe(timeframe)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		WITH bucketed AS (
			SELECT
				date_bin($1::interval, tick_time, TIMESTAMPTZ '2000-01-01 00:00:00+00') AS bucket,
				tick_time,
				price,
				volume
			FROM stock_ticks
			WHERE symbol = UPPER($2)
		),
		agg AS (
			SELECT bucket, MAX(price) AS high, MIN(price) AS low, COALESCE(SUM(volume), 0) AS volume, COUNT(*) AS tick_count
			FROM bucketed
			GROUP BY bucket
		),
		open_p AS (
			SELECT DISTINCT ON (bucket) bucket, price AS open
			FROM bucketed
			ORDER BY bucket, tick_time ASC
		),
		close_p AS (
			SELECT DISTINCT ON (bucket) bucket, price AS close
			FROM bucketed
			ORDER BY bucket, tick_time DESC
		)
		SELECT
			UPPER($2) AS symbol,
			$3 AS timeframe,
			a.bucket,
			o.open,
			a.high,
			a.low,
			c.close,
			a.volume,
			a.tick_count
		FROM agg a
		JOIN open_p o ON o.bucket = a.bucket
		JOIN close_p c ON c.bucket = a.bucket
		ORDER BY a.bucket DESC
		LIMIT $4`, interval, symbol, strings.ToLower(timeframe), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candles := make([]model.StockCandle, 0, limit)
	for rows.Next() {
		var c model.StockCandle
		if err := rows.Scan(&c.Symbol, &c.Timeframe, &c.CandleTime, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume, &c.TickCount); err != nil {
			return nil, err
		}
		candles = append(candles, c)
	}
	return candles, rows.Err()
}

func (s *Store) GetLatestCandlesForSymbols(symbols []string, timeframe string) ([]model.StockCandle, error) {
	if len(symbols) == 0 {
		return []model.StockCandle{}, nil
	}

	unique := make([]string, 0, len(symbols))
	seen := make(map[string]struct{}, len(symbols))
	for _, sym := range symbols {
		s := strings.ToUpper(strings.TrimSpace(sym))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		unique = append(unique, s)
	}
	if len(unique) == 0 {
		return []model.StockCandle{}, nil
	}

	// PostgreSQL does not support QUALIFY; use subquery.
	rows, err := s.db.Query(`
		SELECT symbol, timeframe, candle_time, open, high, low, close, volume, tick_count
		FROM (
			SELECT symbol, timeframe, candle_time, open, high, low, close, volume, tick_count,
				ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY candle_time DESC) AS rn
			FROM stock_candles
			WHERE timeframe = $1 AND symbol = ANY($2)
		) t
		WHERE rn = 1`, strings.ToLower(timeframe), pqArray(unique))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candles := make([]model.StockCandle, 0, len(unique))
	for rows.Next() {
		var c model.StockCandle
		if err := rows.Scan(&c.Symbol, &c.Timeframe, &c.CandleTime, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume, &c.TickCount); err != nil {
			return nil, err
		}
		candles = append(candles, c)
	}
	return candles, rows.Err()
}

func normalizeTimeframe(tf string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(tf)) {
	case "1m":
		return "1 minute", nil
	case "5m":
		return "5 minutes", nil
	case "15m":
		return "15 minutes", nil
	case "1h":
		return "1 hour", nil
	case "1d":
		return "1 day", nil
	default:
		return "", fmt.Errorf("unsupported timeframe: %s", tf)
	}
}

func pqArray(items []string) any {
	return pq.StringArray(items)
}
