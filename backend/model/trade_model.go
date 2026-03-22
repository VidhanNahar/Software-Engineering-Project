package model

import (
	"time"

	"github.com/google/uuid"
)

// TradeOrderRequest is the request payload for buy/sell endpoints.
type TradeOrderRequest struct {
	StockID  uuid.UUID `json:"stock_id"`
	Quantity int       `json:"quantity"`
}

// StockQuote represents current stock market data.
type StockQuote struct {
	StockID           uuid.UUID `json:"stock_id"`
	Name              string    `json:"name"`
	Price             float64   `json:"price"`
	Timestamp         time.Time `json:"timestamp"`
	AvailableQuantity int       `json:"available_quantity"`
}

// PortfolioPosition represents a user holding with live valuation.
type PortfolioPosition struct {
	UserID         uuid.UUID `json:"user_id"`
	StockID        uuid.UUID `json:"stock_id"`
	StockName      string    `json:"stock_name"`
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
