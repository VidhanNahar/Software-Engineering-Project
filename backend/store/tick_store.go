package store

import (
	"backend-go/model"
	"context"
	"database/sql"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type stockSnapshot struct {
	stockID       uuid.UUID
	symbol        string
	price         float64
	previousClose float64
	dayHigh       float64
	dayLow        float64
	totalQty      int64
	totalValue    float64
	totalTrades   int64
}

// SimulateTickCycle performs one centralized price update cycle and stores ticks.
// This is intentionally called only by the market engine.
func (s *Store) SimulateTickCycle(ctx context.Context, now time.Time) ([]model.StockTick, error) {
	isOpen, err := s.IsMarketOpen()
	if err != nil {
		return nil, err
	}
	if !isOpen {
		return []model.StockTick{}, nil
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	snapshots := make([]stockSnapshot, 0, 1024)

	rows, err := tx.QueryContext(ctx, `
	SELECT stock_id, symbol,
		COALESCE(price, previous_close, open_price, close_price, 1) AS price,
		COALESCE(previous_close, price, open_price, close_price, 1) AS previous_close,
		COALESCE(day_high, price, previous_close, open_price, close_price, 1) AS day_high,
		COALESCE(day_low, price, previous_close, open_price, close_price, 1) AS day_low,
		COALESCE(total_traded_qty, 0) AS total_traded_qty,
		COALESCE(total_traded_value, 0) AS total_traded_value,
		COALESCE(total_trades, 0) AS total_trades
	FROM stock
	WHERE price IS NOT NULL
	ORDER BY symbol`)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var s stockSnapshot
		if err := rows.Scan(
			&s.stockID,
			&s.symbol,
			&s.price,
			&s.previousClose,
			&s.dayHigh,
			&s.dayLow,
			&s.totalQty,
			&s.totalValue,
			&s.totalTrades,
		); err != nil {
			rows.Close()
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rng := rand.New(rand.NewSource(now.UnixNano()))
	ticks := make([]model.StockTick, 0, len(snapshots))

	for _, s := range snapshots {
		nextPrice := nextSimulatedPrice(s.price, s.previousClose, s.symbol, rng)
		nextPrice = roundTo(nextPrice, 2)

		if nextPrice == s.price {
			continue
		}

		qty := int64(rng.Intn(50) + 1)

		s.dayHigh = math.Max(s.dayHigh, nextPrice)
		s.dayLow = math.Min(s.dayLow, nextPrice)
		s.totalQty += qty
		s.totalValue += nextPrice * float64(qty)
		s.totalTrades++

		_, err = tx.ExecContext(ctx, `
			UPDATE stock
			SET price = $1, day_high = $2, day_low = $3,
			    total_traded_qty = $4, total_traded_value = $5, total_trades = $6
			WHERE stock_id = $7`,
			nextPrice, s.dayHigh, s.dayLow, s.totalQty, s.totalValue, s.totalTrades, s.stockID)
		if err != nil {
			return nil, err
		}

		tradeValue := nextPrice * float64(qty)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO stock_ticks (stock_id, symbol, tick_time, price, volume, trade_value)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			s.stockID, s.symbol, now, nextPrice, qty, tradeValue)
		if err != nil {
			return nil, err
		}

		if err := upsertOneMinuteCandle(ctx, tx, s.stockID, s.symbol, now, nextPrice, qty); err != nil {
			return nil, err
		}

		ticks = append(ticks, model.StockTick{
			Symbol:     s.symbol,
			TickTime:   now,
			Price:      nextPrice,
			Volume:     qty,
			TradeValue: tradeValue,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return ticks, nil
}

// nextSimulatedPrice uses a smooth random walk with drift + occasional jump.
// This produces realistic demo movement without chaotic spikes.
func nextSimulatedPrice(current float64, previousClose float64, symbol string, rng *rand.Rand) float64 {
	if current <= 0 {
		current = 1
	}
	if previousClose <= 0 {
		previousClose = current
	}

	// dt is 2 seconds over a 6.5 hour trading day.
	dt := 2.0 / (6.5 * 60 * 60)

	baseVol := 0.18 // annualized baseline volatility
	volatility := baseVol + (math.Abs(current-previousClose)/previousClose)*0.5
	if volatility > 0.75 {
		volatility = 0.75
	}

	marketTrendBias := 0.00003
	perStockDrift := stableSymbolDrift(symbol)
	drift := marketTrendBias + perStockDrift

	shock := rng.NormFloat64()
	jump := 0.0
	if rng.Float64() < 0.01 {
		jump = rng.NormFloat64() * 0.005
	}

	logReturn := (drift-0.5*volatility*volatility)*dt + volatility*math.Sqrt(dt)*shock + jump
	next := current * math.Exp(logReturn)

	// Guardrails prevent absurd simulated prices.
	if next < 0.5 {
		next = 0.5
	}
	if next > current*1.08 {
		next = current * 1.08
	}
	if next < current*0.92 {
		next = current * 0.92
	}
	return next
}

func stableSymbolDrift(symbol string) float64 {
	h := 0
	for i := 0; i < len(symbol); i++ {
		h += int(symbol[i])
	}
	normalized := float64((h%21)-10) / 100000.0
	return normalized
}

func roundTo(v float64, places int) float64 {
	p := math.Pow10(places)
	return math.Round(v*p) / p
}

func upsertOneMinuteCandle(ctx context.Context, tx *sql.Tx, stockID uuid.UUID, symbol string, tickTime time.Time, price float64, volume int64) error {
	bucket := tickTime.UTC().Truncate(time.Minute)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO stock_candles (stock_id, symbol, timeframe, candle_time, open, high, low, close, volume, tick_count)
		VALUES ($1, $2, '1m', $3, $4, $4, $4, $4, $5, 1)
		ON CONFLICT (symbol, timeframe, candle_time)
		DO UPDATE SET
			high = GREATEST(stock_candles.high, EXCLUDED.high),
			low = LEAST(stock_candles.low, EXCLUDED.low),
			close = EXCLUDED.close,
			volume = stock_candles.volume + EXCLUDED.volume,
			tick_count = stock_candles.tick_count + 1,
			updated_at = NOW()`,
		stockID, symbol, bucket, price, volume,
	)
	return err
}
