package model

import (
	"time"

	"github.com/google/uuid"
)

// TradeOrderRequest is the request payload for buy/sell endpoints.
type TradeOrderRequest struct {
	StockID       uuid.UUID `json:"stock_id"`
	Quantity      int       `json:"quantity"`
	PricePerStock float64   `json:"price_per_stock"`
	OrderType     string    `json:"order_type"`    // "MARKET" or "LIMIT"
	TimeInForce   string    `json:"time_in_force"` // "DAY", "GTC" (Good Till Canceled)
}

// StockQuote represents current stock market data.
type StockQuote struct {
	StockID           uuid.UUID `json:"stock_id"`
	Symbol            string    `json:"symbol"`
	Name              string    `json:"name"`
	CurrencyCode      string    `json:"currency_code"`
	Country           string    `json:"country"`
	Series            string    `json:"series"`
	ISIN              string    `json:"isin,omitempty"`
	Price             float64   `json:"price"`
	PreviousClose     float64   `json:"previous_close"`
	Open              float64   `json:"open"`
	High              float64   `json:"high"`
	Low               float64   `json:"low"`
	Close             float64   `json:"close"`
	LastTradedPrice   float64   `json:"last_traded_price"`
	Change            float64   `json:"change"`
	ChangePercent     float64   `json:"change_percent"`
	Volume            int64     `json:"volume"`
	TotalTrades       int64     `json:"total_trades"`
	TotalTradedValue  float64   `json:"total_traded_value"`
	TradeDate         time.Time `json:"trade_date"`
	Timestamp         time.Time `json:"timestamp"`
	AvailableQuantity int64     `json:"quantity"`
}

// PortfolioPosition represents a user holding with live valuation.
type PortfolioPosition struct {
	UserID         uuid.UUID `json:"user_id"`
	StockID        uuid.UUID `json:"stock_id"`
	StockName      string    `json:"stock_name"`
	CurrencyCode   string    `json:"currency_code"`
	Quantity       int       `json:"quantity"`
	AvgBuyPrice    float64   `json:"avg_buy_price"`
	CurrentPrice   float64   `json:"current_price"`
	PositionValue  float64   `json:"position_value"`
	LastUpdateTime time.Time `json:"last_update_time"`
}

// OrderRecord is a transaction history entry from orders table.
type OrderRecord struct {
	OrderID       uuid.UUID `json:"order_id"`
	StockID       uuid.UUID `json:"stock_id"`
	UserID        uuid.UUID `json:"user_id"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"`
	Quantity      int       `json:"quantity"`
	PricePerStock float64   `json:"price_per_stock"`
}

type StockHistoryPoint struct {
	TradeDate        time.Time `json:"trade_date"`
	Open             float64   `json:"open"`
	High             float64   `json:"high"`
	Low              float64   `json:"low"`
	Close            float64   `json:"close"`
	Volume           int64     `json:"volume"`
	TotalTrades      int64     `json:"total_trades"`
	TotalTradedValue float64   `json:"total_traded_value"`
}

type AdminStockUpsertRequest struct {
	Symbol           string  `json:"symbol"`
	Name             string  `json:"name"`
	Series           string  `json:"series"`
	ISIN             string  `json:"isin"`
	Price            float64 `json:"price"`
	PreviousClose    float64 `json:"previous_close"`
	Open             float64 `json:"open"`
	High             float64 `json:"high"`
	Low              float64 `json:"low"`
	Close            float64 `json:"close"`
	LastTradedPrice  float64 `json:"last_traded_price"`
	Volume           int64   `json:"volume"`
	TotalTrades      int64   `json:"total_trades"`
	TotalTradedValue float64 `json:"total_traded_value"`
	Quantity         int64   `json:"quantity"`
}

type WatchlistItem struct {
	WatchlistID   uuid.UUID `json:"watchlist_id"`
	StockID       uuid.UUID `json:"stock_id"`
	WatchlistName string    `json:"watchlist_name"`
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	CurrencyCode  string    `json:"currency_code"`
	Price         float64   `json:"price"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"change_percent"`
	Timestamp     time.Time `json:"timestamp"`
}

type AlertRule struct {
	AlertID     uuid.UUID  `json:"alert_id"`
	UserID      uuid.UUID  `json:"user_id"`
	StockID     uuid.UUID  `json:"stock_id"`
	TargetPrice float64    `json:"target_price"`
	Direction   string     `json:"direction"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	TriggeredAt *time.Time `json:"triggered_at,omitempty"`
}

type NotificationItem struct {
	NotificationID uuid.UUID `json:"notification_id"`
	UserID         uuid.UUID `json:"user_id"`
	Type           string    `json:"type"`
	Title          string    `json:"title"`
	Message        string    `json:"message"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
}

type AdminOrderRecord struct {
	OrderID       uuid.UUID `json:"order_id"`
	StockID       uuid.UUID `json:"stock_id"`
	Symbol        string    `json:"symbol"`
	UserID        uuid.UUID `json:"user_id"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"`
	Quantity      int       `json:"quantity"`
	PricePerStock float64   `json:"price_per_stock"`
}
