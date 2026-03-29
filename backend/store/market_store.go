package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// MarketStatus holds market open/close information
type MarketStatus struct {
	MarketID    string     `json:"market_id"`
	IsOpen      bool       `json:"is_open"`
	OpenedAt    *time.Time `json:"opened_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	TotalTrades int64      `json:"total_trades"`
	TotalVolume int64      `json:"total_volume"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// SetCacheValue stores a value in Redis for caching
func (s *Store) SetCacheValue(ctx context.Context, key string, value string, expirySeconds int) error {
	return s.rdb.Set(ctx, key, value, time.Duration(expirySeconds)*time.Second).Err()
}

// GetCacheValue retrieves a value from Redis cache
func (s *Store) GetCacheValue(ctx context.Context, key string) (string, error) {
	return s.rdb.Get(ctx, key).Result()
}

// GetMarketStatus returns current market status
func (s *Store) GetMarketStatus() (*MarketStatus, error) {
	ctx := context.Background()
	cacheKey := "market:status"

	if cached, err := s.GetCacheValue(ctx, cacheKey); err == nil && cached != "" {
		var status MarketStatus
		if jsonErr := json.Unmarshal([]byte(cached), &status); jsonErr == nil {
			return &status, nil
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO market_status (market_id, is_open)
		SELECT gen_random_uuid(), false
		WHERE NOT EXISTS (SELECT 1 FROM market_status LIMIT 1)
	`)
	if err != nil {
		return nil, err
	}

	var status MarketStatus
	var openedAt, closedAt sql.NullTime

	err = s.db.QueryRow(`
		SELECT market_id, is_open, opened_at, closed_at, total_trades, total_volume, created_at, updated_at
		FROM market_status
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(
		&status.MarketID,
		&status.IsOpen,
		&openedAt,
		&closedAt,
		&status.TotalTrades,
		&status.TotalVolume,
		&status.CreatedAt,
		&status.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if openedAt.Valid {
		status.OpenedAt = &openedAt.Time
	}
	if closedAt.Valid {
		status.ClosedAt = &closedAt.Time
	}

	// Cache for 2 seconds (market status changes infrequently)
	if data, jsonErr := json.Marshal(status); jsonErr == nil {
		s.SetCacheValue(ctx, cacheKey, string(data), 2)
	}

	return &status, nil
}

// StartMarket opens the market for trading
func (s *Store) StartMarket() (*MarketStatus, error) {
	ctx := context.Background()
	// First ensure we have a record
	_, err := s.db.Exec(`
		INSERT INTO market_status (market_id, is_open)
		SELECT gen_random_uuid(), false
		WHERE NOT EXISTS (SELECT 1 FROM market_status LIMIT 1)
	`)
	if err != nil {
		return nil, err
	}

	// Now update it
	_, err = s.db.Exec(`
		UPDATE market_status
		SET is_open = true, opened_at = NOW(), closed_at = NULL, updated_at = NOW()
		WHERE market_id = (SELECT market_id FROM market_status LIMIT 1)
	`)
	if err != nil {
		return nil, err
	}

	// Invalidate market status cache
	s.rdb.Del(ctx, "market:status")

	// Fetch and return updated status
	return s.GetMarketStatus()
}

// StopMarket closes the market for trading
func (s *Store) StopMarket() (*MarketStatus, error) {
	ctx := context.Background()
	// First ensure we have a record
	_, err := s.db.Exec(`
		INSERT INTO market_status (market_id, is_open)
		SELECT gen_random_uuid(), false
		WHERE NOT EXISTS (SELECT 1 FROM market_status LIMIT 1)
	`)
	if err != nil {
		return nil, err
	}

	// Now update it
	_, err = s.db.Exec(`
		UPDATE market_status
		SET is_open = false, closed_at = NOW(), updated_at = NOW()
		WHERE market_id = (SELECT market_id FROM market_status LIMIT 1)
	`)
	if err != nil {
		return nil, err
	}

	// Invalidate market status cache
	s.rdb.Del(ctx, "market:status")

	// Fetch and return updated status
	return s.GetMarketStatus()
}

// IsMarketOpen checks if market is currently open
func (s *Store) IsMarketOpen() (bool, error) {
	status, err := s.GetMarketStatus()
	if err != nil {
		return false, err
	}
	if status == nil {
		return false, nil
	}
	return status.IsOpen, nil
}
