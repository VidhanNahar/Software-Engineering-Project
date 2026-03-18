package model

import (
	"time"
)

// TransactionType represents the type of transaction (buy or sell)
type TransactionType string

const (
	TransactionTypeBuy  TransactionType = "buy"
	TransactionTypeSell TransactionType = "sell"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "Pending"
	TransactionStatusFiled     TransactionStatus = "Filed"
	TransactionStatusCancelled TransactionStatus = "Cancelled"
)

// Transaction represents a buy/sell order
type Transaction struct {
	OrderID       string            `json:"order_id"`
	UserID        string            `json:"user_id"`
	StockID       string            `json:"stock_id"`
	StockSymbol   string            `json:"stock_symbol"`
	Type          TransactionType   `json:"type"` // "buy" or "sell"
	Quantity      int               `json:"quantity"`
	PricePerStock float64           `json:"price_per_stock"`
	TotalAmount   float64           `json:"total_amount"`
	Status        TransactionStatus `json:"status"`
	Timestamp     time.Time         `json:"timestamp"`
}

// Stock represents a stock in the market
type Stock struct {
	StockID   string    `json:"stock_id"`
	Symbol    string    `json:"symbol"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Quantity  int       `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}

// Portfolio represents user's stock holdings
type PortfolioHolding struct {
	PortfolioID     string    `json:"portfolio_id"`
	UserID          string    `json:"user_id"`
	StockID         string    `json:"stock_id"`
	StockSymbol     string    `json:"stock_symbol"`
	StockName       string    `json:"stock_name"`
	Quantity        int       `json:"quantity"`
	AverageBuyPrice float64   `json:"average_buy_price"`
	CurrentPrice    float64   `json:"current_price"`
	TotalValue      float64   `json:"total_value"`
	TransactionTime time.Time `json:"transaction_time"`
}

// Wallet represents user's wallet/balance
type Wallet struct {
	WalletID         string  `json:"wallet_id"`
	UserID           string  `json:"user_id"`
	Balance          float64 `json:"balance"`
	LockedBalance    float64 `json:"locked_balance"`
	AvailableBalance float64 `json:"available_balance"`
}

// WatchlistItem represents a stock in user's watchlist
type WatchlistItem struct {
	WatchlistID string    `json:"watchlist_id"`
	UserID      string    `json:"user_id"`
	StockID     string    `json:"stock_id"`
	StockSymbol string    `json:"stock_symbol"`
	StockName   string    `json:"stock_name"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}
