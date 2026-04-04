package store

import (
	"backend-go/model"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var ErrStockNotFound = errors.New("stock not found")
var ErrWatchlistItemNotFound = errors.New("watchlist item not found")
var ErrAlertNotFound = errors.New("alert not found")

// RefreshSimulatedPrices nudges current prices inside each stock's intraday range.
func (s *Store) RefreshSimulatedPrices(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE stock
		SET
			price = ROUND(
				GREATEST(
					COALESCE(day_low, price, previous_close, open_price, close_price, 1),
					LEAST(
						COALESCE(day_high, price, previous_close, open_price, close_price, 1),
						COALESCE(price, previous_close, open_price, close_price, 1)
						+ (
							(random() - 0.5) * GREATEST(
								COALESCE(day_high, price, previous_close, open_price, close_price, 1)
								- COALESCE(day_low, price, previous_close, open_price, close_price, 1),
								COALESCE(price, previous_close, open_price, close_price, 1) * 0.02
							)
						)
					)
				)::numeric,
				2
			),
			timestamp = NOW()
		WHERE price IS NOT NULL;
	`)
	return err
}

// GetStocks returns current market quotes for all stocks.
func (s *Store) GetStocks() ([]model.StockQuote, error) {
	const cacheKey = "stocks:all"
	ctx := context.Background()

	if cached, err := s.GetCacheValue(ctx, cacheKey); err == nil && cached != "" {
		var quotes []model.StockQuote
		if jsonErr := json.Unmarshal([]byte(cached), &quotes); jsonErr == nil {
			return quotes, nil
		}
	}

	rows, err := s.db.Query(`
		SELECT
			stock_id,
			symbol,
			name,
			currency_code,
			country,
			series,
			isin,
			price,
			COALESCE(previous_close, price) AS previous_close,
			COALESCE(open_price, price) AS open_price,
			COALESCE(day_high, price) AS day_high,
			COALESCE(day_low, price) AS day_low,
			COALESCE(close_price, price) AS close_price,
			COALESCE(last_traded_price, price) AS last_traded_price,
			(price - COALESCE(previous_close, price)) AS change,
			CASE
				WHEN COALESCE(previous_close, 0) > 0
				THEN ((price - previous_close) / previous_close) * 100
				ELSE 0
			END AS change_percent,
			COALESCE(total_traded_qty, quantity, 0) AS volume,
			COALESCE(total_trades, 0) AS total_trades,
			COALESCE(total_traded_value, 0) AS total_traded_value,
			COALESCE(trade_date, CURRENT_DATE) AS trade_date,
			timestamp,
			COALESCE(quantity, 0) AS quantity
		FROM stock
		ORDER BY symbol ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotes := make([]model.StockQuote, 0)
	for rows.Next() {
		var quote model.StockQuote
		var isin sql.NullString
		if err := rows.Scan(
			&quote.StockID,
			&quote.Symbol,
			&quote.Name,
			&quote.CurrencyCode,
			&quote.Country,
			&quote.Series,
			&isin,
			&quote.Price,
			&quote.PreviousClose,
			&quote.Open,
			&quote.High,
			&quote.Low,
			&quote.Close,
			&quote.LastTradedPrice,
			&quote.Change,
			&quote.ChangePercent,
			&quote.Volume,
			&quote.TotalTrades,
			&quote.TotalTradedValue,
			&quote.TradeDate,
			&quote.Timestamp,
			&quote.AvailableQuantity,
		); err != nil {
			return nil, err
		}
		if isin.Valid {
			quote.ISIN = isin.String
		}
		quotes = append(quotes, quote)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Cache the result for 5 seconds (frequent updates during market hours)
	if data, jsonErr := json.Marshal(quotes); jsonErr == nil {
		s.SetCacheValue(ctx, cacheKey, string(data), 5)
	}

	return quotes, nil
}

// GetPortfolioByUser returns user holdings with current valuation.
func (s *Store) GetPortfolioByUser(userID uuid.UUID) ([]model.PortfolioPosition, error) {
	rows, err := s.db.Query(`
		SELECT p.user_id, p.stock_id, st.name, st.currency_code, p.quantity, p.price, st.price, (p.quantity * st.price), p.transaction_time
		FROM portfolio p
		INNER JOIN stock st ON st.stock_id = p.stock_id
		WHERE p.user_id = $1
		ORDER BY p.transaction_time DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make([]model.PortfolioPosition, 0)
	for rows.Next() {
		var position model.PortfolioPosition
		if err := rows.Scan(
			&position.UserID,
			&position.StockID,
			&position.StockName,
			&position.CurrencyCode,
			&position.Quantity,
			&position.AvgBuyPrice,
			&position.CurrentPrice,
			&position.PositionValue,
			&position.LastUpdateTime,
		); err != nil {
			return nil, err
		}
		positions = append(positions, position)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return positions, nil
}

// GetOrdersByUser returns transaction history in reverse chronology.
func (s *Store) GetOrdersByUser(userID uuid.UUID) ([]model.OrderRecord, error) {
	rows, err := s.db.Query(`
		SELECT order_id, stock_id, user_id, timestamp, status, quantity, price_per_stock
		FROM orders
		WHERE user_id = $1
		ORDER BY timestamp DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.OrderRecord, 0)
	for rows.Next() {
		var order model.OrderRecord
		if err := rows.Scan(&order.OrderID, &order.StockID, &order.UserID, &order.Timestamp, &order.Status, &order.Quantity, &order.PricePerStock); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *Store) GetStockByID(stockID uuid.UUID) (*model.StockQuote, error) {
	cacheKey := fmt.Sprintf("stock:id:%s", stockID.String())
	ctx := context.Background()

	if cached, err := s.GetCacheValue(ctx, cacheKey); err == nil && cached != "" {
		var quote model.StockQuote
		if jsonErr := json.Unmarshal([]byte(cached), &quote); jsonErr == nil {
			return &quote, nil
		}
	}

	row := s.db.QueryRow(`
		SELECT
			stock_id,
			symbol,
			name,
			currency_code,
			country,
			series,
			isin,
			price,
			COALESCE(previous_close, price) AS previous_close,
			COALESCE(open_price, price) AS open_price,
			COALESCE(day_high, price) AS day_high,
			COALESCE(day_low, price) AS day_low,
			COALESCE(close_price, price) AS close_price,
			COALESCE(last_traded_price, price) AS last_traded_price,
			(price - COALESCE(previous_close, price)) AS change,
			CASE
				WHEN COALESCE(previous_close, 0) > 0
				THEN ((price - previous_close) / previous_close) * 100
				ELSE 0
			END AS change_percent,
			COALESCE(total_traded_qty, quantity, 0) AS volume,
			COALESCE(total_trades, 0) AS total_trades,
			COALESCE(total_traded_value, 0) AS total_traded_value,
			COALESCE(trade_date, CURRENT_DATE) AS trade_date,
			timestamp,
			COALESCE(quantity, 0) AS quantity
		FROM stock
		WHERE stock_id = $1`, stockID)

	var quote model.StockQuote
	var isin sql.NullString
	err := row.Scan(
		&quote.StockID,
		&quote.Symbol,
		&quote.Name,
		&quote.CurrencyCode,
		&quote.Country,
		&quote.Series,
		&isin,
		&quote.Price,
		&quote.PreviousClose,
		&quote.Open,
		&quote.High,
		&quote.Low,
		&quote.Close,
		&quote.LastTradedPrice,
		&quote.Change,
		&quote.ChangePercent,
		&quote.Volume,
		&quote.TotalTrades,
		&quote.TotalTradedValue,
		&quote.TradeDate,
		&quote.Timestamp,
		&quote.AvailableQuantity,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStockNotFound
		}
		return nil, err
	}
	if isin.Valid {
		quote.ISIN = isin.String
	}

	// Cache for 5 seconds
	if data, jsonErr := json.Marshal(quote); jsonErr == nil {
		s.SetCacheValue(ctx, cacheKey, string(data), 5)
	}

	return &quote, nil
}

func (s *Store) GetStockBySymbol(symbol string) (*model.StockQuote, error) {
	cacheKey := fmt.Sprintf("stock:symbol:%s", strings.ToUpper(symbol))
	ctx := context.Background()

	if cached, err := s.GetCacheValue(ctx, cacheKey); err == nil && cached != "" {
		var quote model.StockQuote
		if jsonErr := json.Unmarshal([]byte(cached), &quote); jsonErr == nil {
			return &quote, nil
		}
	}

	row := s.db.QueryRow(`SELECT stock_id FROM stock WHERE symbol = UPPER($1)`, symbol)
	var stockID uuid.UUID
	if err := row.Scan(&stockID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStockNotFound
		}
		return nil, err
	}

	quote, err := s.GetStockByID(stockID)
	if err != nil {
		return nil, err
	}

	// Cache for 5 seconds
	if data, jsonErr := json.Marshal(quote); jsonErr == nil {
		s.SetCacheValue(ctx, cacheKey, string(data), 5)
	}

	return quote, nil
}

func (s *Store) SearchStocks(query string) ([]model.StockQuote, error) {
	cacheKey := fmt.Sprintf("stocks:search:%s", strings.ToLower(query))
	ctx := context.Background()

	if cached, err := s.GetCacheValue(ctx, cacheKey); err == nil && cached != "" {
		var quotes []model.StockQuote
		if jsonErr := json.Unmarshal([]byte(cached), &quotes); jsonErr == nil {
			return quotes, nil
		}
	}

	rows, err := s.db.Query(`
		SELECT
			stock_id,
			symbol,
			name,
			currency_code,
			country,
			series,
			isin,
			price,
			COALESCE(previous_close, price) AS previous_close,
			COALESCE(open_price, price) AS open_price,
			COALESCE(day_high, price) AS day_high,
			COALESCE(day_low, price) AS day_low,
			COALESCE(close_price, price) AS close_price,
			COALESCE(last_traded_price, price) AS last_traded_price,
			(price - COALESCE(previous_close, price)) AS change,
			CASE
				WHEN COALESCE(previous_close, 0) > 0
				THEN ((price - previous_close) / previous_close) * 100
				ELSE 0
			END AS change_percent,
			COALESCE(total_traded_qty, quantity, 0) AS volume,
			COALESCE(total_trades, 0) AS total_trades,
			COALESCE(total_traded_value, 0) AS total_traded_value,
			COALESCE(trade_date, CURRENT_DATE) AS trade_date,
			timestamp,
			COALESCE(quantity, 0) AS quantity
		FROM stock
		WHERE symbol ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%'
		ORDER BY symbol ASC
		LIMIT 50`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotes := make([]model.StockQuote, 0)
	for rows.Next() {
		var quote model.StockQuote
		var isin sql.NullString
		if err := rows.Scan(
			&quote.StockID,
			&quote.Symbol,
			&quote.Name,
			&quote.CurrencyCode,
			&quote.Country,
			&quote.Series,
			&isin,
			&quote.Price,
			&quote.PreviousClose,
			&quote.Open,
			&quote.High,
			&quote.Low,
			&quote.Close,
			&quote.LastTradedPrice,
			&quote.Change,
			&quote.ChangePercent,
			&quote.Volume,
			&quote.TotalTrades,
			&quote.TotalTradedValue,
			&quote.TradeDate,
			&quote.Timestamp,
			&quote.AvailableQuantity,
		); err != nil {
			return nil, err
		}
		if isin.Valid {
			quote.ISIN = isin.String
		}
		quotes = append(quotes, quote)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Cache search results for 10 seconds (less frequent updates)
	if data, jsonErr := json.Marshal(quotes); jsonErr == nil {
		s.SetCacheValue(ctx, cacheKey, string(data), 10)
	}

	return quotes, nil
}

func (s *Store) GetStockHistory(stockID uuid.UUID, limit int) ([]model.StockHistoryPoint, error) {
	if limit <= 0 {
		limit = 120
	}

	rows, err := s.db.Query(`
		SELECT trade_date, open_price, day_high, day_low, close_price, total_traded_qty, total_trades, total_traded_value
		FROM stock_daily_data
		WHERE stock_id = $1
		ORDER BY trade_date DESC
		LIMIT $2`, stockID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]model.StockHistoryPoint, 0)
	for rows.Next() {
		var point model.StockHistoryPoint
		if err := rows.Scan(
			&point.TradeDate,
			&point.Open,
			&point.High,
			&point.Low,
			&point.Close,
			&point.Volume,
			&point.TotalTrades,
			&point.TotalTradedValue,
		); err != nil {
			return nil, err
		}
		history = append(history, point)
	}
	return history, rows.Err()
}

func (s *Store) AdminCreateStock(req model.AdminStockUpsertRequest) error {
	ctx := context.Background()
	_, err := s.db.Exec(`
		INSERT INTO stock (
			symbol, name, series, isin, price, previous_close, open_price, day_high, day_low, close_price,
			last_traded_price, total_traded_qty, total_trades, total_traded_value, quantity, trade_date, timestamp
		) VALUES (
			UPPER($1), $2, UPPER($3), NULLIF($4, ''), $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, CURRENT_DATE, NOW()
		)
		ON CONFLICT (symbol) DO UPDATE SET
			name = EXCLUDED.name,
			series = EXCLUDED.series,
			isin = EXCLUDED.isin,
			price = EXCLUDED.price,
			previous_close = EXCLUDED.previous_close,
			open_price = EXCLUDED.open_price,
			day_high = EXCLUDED.day_high,
			day_low = EXCLUDED.day_low,
			close_price = EXCLUDED.close_price,
			last_traded_price = EXCLUDED.last_traded_price,
			total_traded_qty = EXCLUDED.total_traded_qty,
			total_trades = EXCLUDED.total_trades,
			total_traded_value = EXCLUDED.total_traded_value,
			quantity = EXCLUDED.quantity,
			trade_date = EXCLUDED.trade_date,
			timestamp = NOW()`,
		req.Symbol,
		req.Name,
		req.Series,
		req.ISIN,
		req.Price,
		req.PreviousClose,
		req.Open,
		req.High,
		req.Low,
		req.Close,
		req.LastTradedPrice,
		req.Volume,
		req.TotalTrades,
		req.TotalTradedValue,
		req.Quantity,
	)
	if err != nil {
		return err
	}

	// Invalidate relevant caches
	s.rdb.Del(ctx, "stocks:all", "stocks:top:50", "stocks:top:100", fmt.Sprintf("stock:symbol:%s", strings.ToUpper(req.Symbol)))
	return nil
}

func (s *Store) AdminUpdateStock(stockID uuid.UUID, req model.AdminStockUpsertRequest) error {
	ctx := context.Background()
	res, err := s.db.Exec(`
		UPDATE stock
		SET symbol = UPPER($1),
			name = $2,
			series = UPPER($3),
			isin = NULLIF($4, ''),
			price = $5,
			previous_close = $6,
			open_price = $7,
			day_high = $8,
			day_low = $9,
			close_price = $10,
			last_traded_price = $11,
			total_traded_qty = $12,
			total_trades = $13,
			total_traded_value = $14,
			quantity = $15,
			trade_date = CURRENT_DATE,
			timestamp = NOW()
		WHERE stock_id = $16`,
		req.Symbol,
		req.Name,
		req.Series,
		req.ISIN,
		req.Price,
		req.PreviousClose,
		req.Open,
		req.High,
		req.Low,
		req.Close,
		req.LastTradedPrice,
		req.Volume,
		req.TotalTrades,
		req.TotalTradedValue,
		req.Quantity,
		stockID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrStockNotFound
	}

	// Invalidate relevant caches
	s.rdb.Del(ctx, "stocks:all", "stocks:top:50", "stocks:top:100", fmt.Sprintf("stock:id:%s", stockID.String()), fmt.Sprintf("stock:symbol:%s", strings.ToUpper(req.Symbol)))
	return nil
}

func (s *Store) AdminDeleteStock(stockID uuid.UUID) error {
	ctx := context.Background()
	// Fetch stock symbol before deletion for cache invalidation
	var symbol string
	err := s.db.QueryRow(`SELECT symbol FROM stock WHERE stock_id = $1`, stockID).Scan(&symbol)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrStockNotFound
		}
		return err
	}

	res, err := s.db.Exec(`DELETE FROM stock WHERE stock_id = $1`, stockID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrStockNotFound
	}

	// Invalidate relevant caches
	s.rdb.Del(ctx, "stocks:all", "stocks:top:50", "stocks:top:100", fmt.Sprintf("stock:id:%s", stockID.String()), fmt.Sprintf("stock:symbol:%s", strings.ToUpper(symbol)))
	return nil
}

func (s *Store) GetTopStocks(limit int) ([]model.StockQuote, error) {
	if limit <= 0 {
		limit = 50
	}

	cacheKey := fmt.Sprintf("stocks:top:%d", limit)
	ctx := context.Background()

	if cached, err := s.GetCacheValue(ctx, cacheKey); err == nil && cached != "" {
		var quotes []model.StockQuote
		if jsonErr := json.Unmarshal([]byte(cached), &quotes); jsonErr == nil {
			return quotes, nil
		}
	}

	rows, err := s.db.Query(`
		SELECT
			stock_id,
			symbol,
			name,
			currency_code,
			country,
			series,
			isin,
			price,
			COALESCE(previous_close, price) AS previous_close,
			COALESCE(open_price, price) AS open_price,
			COALESCE(day_high, price) AS day_high,
			COALESCE(day_low, price) AS day_low,
			COALESCE(close_price, price) AS close_price,
			COALESCE(last_traded_price, price) AS last_traded_price,
			(price - COALESCE(previous_close, price)) AS change,
			CASE
				WHEN COALESCE(previous_close, 0) > 0
				THEN ((price - previous_close) / previous_close) * 100
				ELSE 0
			END AS change_percent,
			COALESCE(total_traded_qty, quantity, 0) AS volume,
			COALESCE(total_trades, 0) AS total_trades,
			COALESCE(total_traded_value, 0) AS total_traded_value,
			COALESCE(trade_date, CURRENT_DATE) AS trade_date,
			timestamp,
			COALESCE(quantity, 0) AS quantity
		FROM stock
		ORDER BY COALESCE(total_traded_value, price * quantity, 0) DESC, symbol ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotes := make([]model.StockQuote, 0)
	for rows.Next() {
		var quote model.StockQuote
		var isin sql.NullString
		if err := rows.Scan(
			&quote.StockID,
			&quote.Symbol,
			&quote.Name,
			&quote.CurrencyCode,
			&quote.Country,
			&quote.Series,
			&isin,
			&quote.Price,
			&quote.PreviousClose,
			&quote.Open,
			&quote.High,
			&quote.Low,
			&quote.Close,
			&quote.LastTradedPrice,
			&quote.Change,
			&quote.ChangePercent,
			&quote.Volume,
			&quote.TotalTrades,
			&quote.TotalTradedValue,
			&quote.TradeDate,
			&quote.Timestamp,
			&quote.AvailableQuantity,
		); err != nil {
			return nil, err
		}
		if isin.Valid {
			quote.ISIN = isin.String
		}
		quotes = append(quotes, quote)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Cache for 5 seconds
	if data, jsonErr := json.Marshal(quotes); jsonErr == nil {
		s.SetCacheValue(ctx, cacheKey, string(data), 5)
	}

	return quotes, rows.Err()
}

func (s *Store) GetAllOrders(limit int) ([]model.AdminOrderRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(`
		SELECT o.order_id, o.stock_id, st.symbol, o.user_id, o.timestamp, o.status, o.quantity, o.price_per_stock
		FROM orders o
		INNER JOIN stock st ON st.stock_id = o.stock_id
		ORDER BY o.timestamp DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.AdminOrderRecord, 0)
	for rows.Next() {
		var order model.AdminOrderRecord
		if err := rows.Scan(
			&order.OrderID,
			&order.StockID,
			&order.Symbol,
			&order.UserID,
			&order.Timestamp,
			&order.Status,
			&order.Quantity,
			&order.PricePerStock,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (s *Store) GetWatchlistByUser(userID uuid.UUID) ([]model.WatchlistItem, error) {
	rows, err := s.db.Query(`
		SELECT
			w.watchlist_id,
			w.stock_id,
			w.watchlist_name,
			st.symbol,
			st.name,
			st.currency_code,
			st.price,
			(st.price - COALESCE(st.previous_close, st.price)) AS change,
			CASE
				WHEN COALESCE(st.previous_close, 0) > 0
				THEN ((st.price - st.previous_close) / st.previous_close) * 100
				ELSE 0
			END AS change_percent,
			w.timestamp
		FROM watchlist w
		INNER JOIN stock st ON st.stock_id = w.stock_id
		WHERE w.user_id = $1
		ORDER BY w.timestamp DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	watchlist := make([]model.WatchlistItem, 0)
	for rows.Next() {
		var item model.WatchlistItem
		if err := rows.Scan(
			&item.WatchlistID,
			&item.StockID,
			&item.WatchlistName,
			&item.Symbol,
			&item.Name,
			&item.CurrencyCode,
			&item.Price,
			&item.Change,
			&item.ChangePercent,
			&item.Timestamp,
		); err != nil {
			return nil, err
		}
		watchlist = append(watchlist, item)
	}
	return watchlist, rows.Err()
}

func (s *Store) AddWatchlistItem(userID, stockID uuid.UUID, watchlistName string) (uuid.UUID, error) {
	if watchlistName == "" {
		watchlistName = "Default"
	}

	var existingID uuid.UUID
	err := s.db.QueryRow(`
		SELECT watchlist_id
		FROM watchlist
		WHERE user_id = $1 AND stock_id = $2
		ORDER BY timestamp DESC
		LIMIT 1`, userID, stockID).Scan(&existingID)
	if err == nil {
		return existingID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, err
	}

	var watchlistID uuid.UUID
	err = s.db.QueryRow(`
		INSERT INTO watchlist (user_id, watchlist_name, stock_id, quantity, price, timestamp)
		VALUES ($1, $2, $3, 1, 1, NOW())
		RETURNING watchlist_id`, userID, watchlistName, stockID).Scan(&watchlistID)
	if err != nil {
		return uuid.Nil, err
	}
	return watchlistID, nil
}

func (s *Store) RemoveWatchlistItem(userID, watchlistID uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM watchlist WHERE watchlist_id = $1 AND user_id = $2`, watchlistID, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWatchlistItemNotFound
	}
	return nil
}

func (s *Store) CreateAlert(userID, stockID uuid.UUID, targetPrice float64, direction string) (*model.AlertRule, error) {
	row := s.db.QueryRow(`
		INSERT INTO alerts (user_id, stock_id, target_price, direction)
		VALUES ($1, $2, $3, $4)
		RETURNING alert_id, user_id, stock_id, target_price, direction, is_active, created_at, triggered_at`, userID, stockID, targetPrice, direction)

	var alert model.AlertRule
	if err := row.Scan(
		&alert.AlertID,
		&alert.UserID,
		&alert.StockID,
		&alert.TargetPrice,
		&alert.Direction,
		&alert.IsActive,
		&alert.CreatedAt,
		&alert.TriggeredAt,
	); err != nil {
		return nil, err
	}
	return &alert, nil
}

func (s *Store) GetAlertsByUser(userID uuid.UUID) ([]model.AlertRule, error) {
	rows, err := s.db.Query(`
		SELECT alert_id, user_id, stock_id, target_price, direction, is_active, created_at, triggered_at
		FROM alerts
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]model.AlertRule, 0)
	for rows.Next() {
		var alert model.AlertRule
		if err := rows.Scan(
			&alert.AlertID,
			&alert.UserID,
			&alert.StockID,
			&alert.TargetPrice,
			&alert.Direction,
			&alert.IsActive,
			&alert.CreatedAt,
			&alert.TriggeredAt,
		); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}

func (s *Store) DeleteAlert(userID, alertID uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM alerts WHERE alert_id = $1 AND user_id = $2`, alertID, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAlertNotFound
	}
	return nil
}

func (s *Store) CreateNotification(userID uuid.UUID, typ, title, message string) error {
	_, err := s.db.Exec(`
		INSERT INTO notifications (user_id, type, title, message)
		VALUES ($1, $2, $3, $4)`, userID, typ, title, message)
	return err
}

func (s *Store) GetNotificationsByUser(userID uuid.UUID, unreadOnly bool) ([]model.NotificationItem, error) {
	query := `
		SELECT notification_id, user_id, type, title, message, is_read, created_at
		FROM notifications
		WHERE user_id = $1`
	args := []any{userID}
	if unreadOnly {
		query += ` AND is_read = false`
	}
	query += ` ORDER BY created_at DESC LIMIT 200`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.NotificationItem, 0)
	for rows.Next() {
		var n model.NotificationItem
		if err := rows.Scan(
			&n.NotificationID,
			&n.UserID,
			&n.Type,
			&n.Title,
			&n.Message,
			&n.IsRead,
			&n.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

func (s *Store) MarkNotificationRead(userID, notificationID uuid.UUID) error {
	_, err := s.db.Exec(`
		UPDATE notifications
		SET is_read = true
		WHERE notification_id = $1 AND user_id = $2`, notificationID, userID)
	return err
}

func (s *Store) EvaluateAlerts() error {
	_, err := s.db.Exec(`
		WITH triggered AS (
			UPDATE alerts a
			SET is_active = false,
				triggered_at = NOW()
			FROM stock s
			WHERE a.stock_id = s.stock_id
				AND a.is_active = true
				AND (
					(a.direction = 'above' AND s.price >= a.target_price)
					OR
					(a.direction = 'below' AND s.price <= a.target_price)
				)
			RETURNING a.user_id, s.symbol, a.target_price, a.direction
		)
		INSERT INTO notifications (user_id, type, title, message)
		SELECT
			t.user_id,
			'alert',
			'Price alert triggered',
			'Stock ' || t.symbol || ' crossed your ' || t.direction || ' target of ' || t.target_price::text
		FROM triggered t;
	`)
	return err
}
