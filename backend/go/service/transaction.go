package service

import (
	"context"
	"fmt"
	"time"

	"backend-go/db"
	"backend-go/model"
)

// BuyStock processes a buy transaction for a user
func BuyStock(ctx context.Context, userID string, stockID string, quantity int, currentPrice float64) (*model.Transaction, error) {
	totalCost := float64(quantity) * currentPrice

	// Start transaction
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get user's wallet
	var balance, lockedBalance float64
	err = tx.QueryRow(ctx,
		`SELECT balance, locked_balance FROM wallet WHERE user_id = $1 FOR UPDATE`,
		userID,
	).Scan(&balance, &lockedBalance)
	if err != nil {
		return nil, fmt.Errorf("wallet not found: %w", err)
	}

	availableBalance := balance - lockedBalance
	if availableBalance < totalCost {
		return nil, fmt.Errorf("insufficient balance: required %.2f, available %.2f", totalCost, availableBalance)
	}

	// Create order record
	var orderID string
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (stock_id, user_id, timestamp, status, quantity, price_per_stock)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING order_id`,
		stockID, userID, time.Now(), model.TransactionStatusFiled, quantity, currentPrice,
	).Scan(&orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Update wallet - lock the balance
	_, err = tx.Exec(ctx,
		`UPDATE wallet SET locked_balance = locked_balance + $1 WHERE user_id = $2`,
		totalCost, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update wallet: %w", err)
	}

	// Check if user already holds this stock
	var existingQty int
	var portfolioID string
	err = tx.QueryRow(ctx,
		`SELECT portfolio_id, quantity FROM portfolio
		 WHERE user_id = $1 AND stock_id = $2
		 ORDER BY transaction_time DESC LIMIT 1`,
		userID, stockID,
	).Scan(&portfolioID, &existingQty)

	if err == nil {
		// Update existing portfolio entry
		_, err = tx.Exec(ctx,
			`INSERT INTO portfolio (portfolio_id, user_id, stock_id, transaction_time, price, quantity)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			portfolioID, userID, stockID, time.Now(), currentPrice, quantity,
		)
	} else {
		// Create new portfolio entry
		_, err = tx.Exec(ctx,
			`INSERT INTO portfolio (user_id, stock_id, transaction_time, price, quantity)
			 VALUES ($1, $2, $3, $4, $5)`,
			userID, stockID, time.Now(), currentPrice, quantity,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to update portfolio: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	transaction := &model.Transaction{
		OrderID:       orderID,
		UserID:        userID,
		StockID:       stockID,
		Type:          model.TransactionTypeBuy,
		Quantity:      quantity,
		PricePerStock: currentPrice,
		TotalAmount:   totalCost,
		Status:        model.TransactionStatusFiled,
		Timestamp:     time.Now(),
	}

	return transaction, nil
}

// SellStock processes a sell transaction for a user
func SellStock(ctx context.Context, userID string, stockID string, quantity int, currentPrice float64) (*model.Transaction, error) {
	totalProceeds := float64(quantity) * currentPrice

	// Start transaction
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get user's current holdings
	var holdingQty int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity), 0) FROM portfolio
		 WHERE user_id = $1 AND stock_id = $2`,
		userID, stockID,
	).Scan(&holdingQty)
	if err != nil {
		return nil, fmt.Errorf("failed to check holdings: %w", err)
	}

	if holdingQty < quantity {
		return nil, fmt.Errorf("insufficient stock holdings: required %d, available %d", quantity, holdingQty)
	}

	// Create order record
	var orderID string
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (stock_id, user_id, timestamp, status, quantity, price_per_stock)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING order_id`,
		stockID, userID, time.Now(), model.TransactionStatusFiled, quantity, currentPrice,
	).Scan(&orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Update wallet - add proceeds
	_, err = tx.Exec(ctx,
		`UPDATE wallet SET balance = balance + $1 WHERE user_id = $2`,
		totalProceeds, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update wallet: %w", err)
	}

	// Record the sell in portfolio (negative quantity)
	_, err = tx.Exec(ctx,
		`INSERT INTO portfolio (user_id, stock_id, transaction_time, price, quantity)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, stockID, time.Now(), currentPrice, -quantity,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record sell: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	transaction := &model.Transaction{
		OrderID:       orderID,
		UserID:        userID,
		StockID:       stockID,
		Type:          model.TransactionTypeSell,
		Quantity:      quantity,
		PricePerStock: currentPrice,
		TotalAmount:   totalProceeds,
		Status:        model.TransactionStatusFiled,
		Timestamp:     time.Now(),
	}

	return transaction, nil
}

// GetUserPortfolio retrieves all stocks held by a user
func GetUserPortfolio(ctx context.Context, userID string) ([]model.PortfolioHolding, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT DISTINCT
			p.portfolio_id,
			p.user_id,
			p.stock_id,
			s.symbol,
			s.name,
			COALESCE(SUM(p.quantity), 0),
			AVG(p.price),
			s.price,
			p.transaction_time
		 FROM portfolio p
		 JOIN stock s ON p.stock_id = s.stock_id
		 WHERE p.user_id = $1
		 GROUP BY p.stock_id, p.portfolio_id, p.user_id, s.symbol, s.name, s.price, p.transaction_time
		 HAVING COALESCE(SUM(p.quantity), 0) > 0`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch portfolio: %w", err)
	}
	defer rows.Close()

	var holdings []model.PortfolioHolding
	for rows.Next() {
		var holding model.PortfolioHolding
		err := rows.Scan(
			&holding.PortfolioID,
			&holding.UserID,
			&holding.StockID,
			&holding.StockSymbol,
			&holding.StockName,
			&holding.Quantity,
			&holding.AverageBuyPrice,
			&holding.CurrentPrice,
			&holding.TransactionTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan portfolio: %w", err)
		}
		holding.TotalValue = float64(holding.Quantity) * holding.CurrentPrice
		holdings = append(holdings, holding)
	}

	return holdings, nil
}

// GetUserWallet retrieves the user's wallet information
func GetUserWallet(ctx context.Context, userID string) (*model.Wallet, error) {
	var wallet model.Wallet
	err := db.Pool.QueryRow(ctx,
		`SELECT wallet_id, user_id, balance, locked_balance FROM wallet WHERE user_id = $1`,
		userID,
	).Scan(&wallet.WalletID, &wallet.UserID, &wallet.Balance, &wallet.LockedBalance)

	if err != nil {
		return nil, fmt.Errorf("wallet not found: %w", err)
	}

	wallet.AvailableBalance = wallet.Balance - wallet.LockedBalance
	return &wallet, nil
}

// GetTransactionHistory retrieves all transactions for a user
func GetTransactionHistory(ctx context.Context, userID string, limit int, offset int) ([]model.Transaction, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT order_id, user_id, stock_id, timestamp, status, quantity, price_per_stock
		 FROM orders
		 WHERE user_id = $1
		 ORDER BY timestamp DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	defer rows.Close()

	var transactions []model.Transaction
	for rows.Next() {
		var t model.Transaction
		err := rows.Scan(
			&t.OrderID,
			&t.UserID,
			&t.StockID,
			&t.Timestamp,
			&t.Status,
			&t.Quantity,
			&t.PricePerStock,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		t.TotalAmount = float64(t.Quantity) * t.PricePerStock
		transactions = append(transactions, t)
	}

	return transactions, nil
}

// InitializeWallet creates a wallet for a new user
func InitializeWallet(ctx context.Context, userID string, initialBalance float64) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO wallet (user_id, balance, locked_balance) VALUES ($1, $2, 0)`,
		userID, initialBalance,
	)
	return err
}
