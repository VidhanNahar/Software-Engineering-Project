package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

func getConversionRate(currency string) float64 {
	if strings.ToUpper(currency) == "USD" {
		return 83.5
	}
	return 1.0
}

var (
	// Returned when wallet balance is lower than buy amount.
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	// Returned when user tries to sell more shares than owned.
	ErrInsufficientShares = errors.New("insufficient shares in portfolio")
	// Returned when wallet row is missing for a valid user.
	ErrWalletNotFound = errors.New("wallet not found")
)

// TradeRequest carries minimum data required for buy/sell execution.
type TradeRequest struct {
	UserID        uuid.UUID
	StockID       uuid.UUID
	Quantity      int
	PricePerStock float64
}

// ExecuteBuyTx performs an atomic buy trade:
// debit wallet, update portfolio, and write order history.
func (s *Store) ExecuteBuyTx(ctx context.Context, req TradeRequest) error {
	// Serializable isolation reduces race conditions in concurrent trades.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	// Rollback is harmless after commit, and protects early returns.
	defer tx.Rollback()

	// Basic input guard before hitting DB.
	if req.Quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}
	if req.PricePerStock <= 0 {
		return errors.New("price per stock must be greater than 0")
	}

	// Get currency code and validate stock exists
	var currencyCode string
	if err = tx.QueryRowContext(ctx, `SELECT currency_code FROM stock WHERE stock_id = $1`, req.StockID).Scan(&currencyCode); err != nil {
		return err
	}

	// Use the price sent from frontend (real-time price user saw when clicking buy)
	stockPrice := req.PricePerStock

	conversionRate := getConversionRate(currencyCode)
	totalCost := stockPrice * float64(req.Quantity) * conversionRate

	log.Printf("💰 [BUY] User: %s, Stock: %s, Qty: %d, Price: %.2f, Currency: %s, Rate: %.2f, Total Cost: %.2f",
		req.UserID, req.StockID, req.Quantity, stockPrice, currencyCode, conversionRate, totalCost)

	// Lock wallet row to prevent double-spend in concurrent buys.
	var balance float64
	err = tx.QueryRowContext(ctx, `SELECT balance from wallet WHERE user_id=$1 FOR UPDATE`, req.UserID).Scan(&balance)
	if err != nil {
		return err
	}
	if balance < totalCost {
		return fmt.Errorf("insufficient balance: have %.2f, need %.2f", balance, totalCost)
	}

	// Debit wallet only after passing balance validation.
	walletRes, err := tx.ExecContext(ctx, `UPDATE wallet SET balance = balance - $1 WHERE user_id=$2`, totalCost, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to update wallet: %w", err)
	}
	walletRows, err := walletRes.RowsAffected()
	if err != nil {
		return err
	}
	if walletRows == 0 {
		return fmt.Errorf("wallet not found for user: %s", req.UserID)
	}

	// Log AFTER balance
	var afterBalance float64
	err = tx.QueryRowContext(ctx, `SELECT balance FROM wallet WHERE user_id = $1`, req.UserID).Scan(&afterBalance)
	if err != nil {
		log.Printf("⚠️ [BUY] Failed to read balance after update: %v", err)
	}

	log.Printf("✅ [BUY] Wallet updated. BEFORE: %.2f, AFTER: %.2f, DIFFERENCE: %.2f", balance, afterBalance, balance-afterBalance)

	now := time.Now()

	// Calculate the correct average price when buying same stock multiple times
	var existingQty int
	var existingPrice float64

	err = tx.QueryRowContext(ctx, `SELECT quantity, price FROM portfolio WHERE user_id = $1 AND stock_id = $2`,
		req.UserID, req.StockID).Scan(&existingQty, &existingPrice)
	if err == nil {
		// Stock already exists in portfolio - calculate new average price
		totalCost := existingPrice*float64(existingQty) + stockPrice*float64(req.Quantity)
		newQuantity := existingQty + req.Quantity
		avgPrice := totalCost / float64(newQuantity)

		log.Printf("📊 [BUY-AVG] Old: Qty=%d, Price=%.2f | New: Qty=%d, Price=%.2f | Result: Qty=%d, AvgPrice=%.2f",
			existingQty, existingPrice, req.Quantity, stockPrice, newQuantity, avgPrice)

		// Update with correct average price
		res, err := tx.ExecContext(ctx, `UPDATE portfolio
									SET quantity = quantity + $1, transaction_time = $2, price = $3
									WHERE user_id = $4 AND stock_id = $5`,
			req.Quantity, now, avgPrice, req.UserID, req.StockID)
		if err != nil {
			return err
		}
		_, err = res.RowsAffected()
		if err != nil {
			return err
		}
	} else if err == sql.ErrNoRows {
		// First buy for this stock
		log.Printf("📊 [BUY-NEW] Creating new portfolio entry: Qty=%d, Price=%.2f", req.Quantity, stockPrice)
		_, err = tx.ExecContext(ctx, `INSERT INTO portfolio (user_id, stock_id, transaction_time, price, quantity)
  								VALUES ($1, $2, $3, $4, $5)`, req.UserID, req.StockID, now, stockPrice, req.Quantity)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	// Orders table is immutable trade history.
	_, err = tx.ExecContext(ctx, `INSERT INTO orders (stock_id, user_id, timestamp, status, quantity, price_per_stock)
		VALUES ($1, $2, $3, $4, $5, $6)`, req.StockID, req.UserID, now, "Filed", req.Quantity, stockPrice)
	if err != nil {
		return err
	}

	// Best-effort transaction confirmation notification.
	_, _ = tx.ExecContext(ctx, `
		INSERT INTO notifications (user_id, type, title, message)
		VALUES ($1, $2, $3, $4)`,
		req.UserID,
		"trade",
		"Buy order executed",
		"Your buy order was executed successfully.",
	)

	// Commit makes all buy changes visible together.
	return tx.Commit()
}

// ExecuteSellTx performs an atomic sell trade:
// reduce holdings, credit wallet, and write order history.
func (s *Store) ExecuteSellTx(ctx context.Context, req TradeRequest) error {
	// Serializable isolation reduces race conditions in concurrent trades.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	// Rollback is harmless after commit, and protects early returns.
	defer tx.Rollback()

	// Basic input guard before hitting DB.
	if req.Quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}
	if req.PricePerStock <= 0 {
		return errors.New("price per stock must be greater than 0")
	}

	// Get currency code and validate stock exists
	var currencyCode string
	if err = tx.QueryRowContext(ctx, `SELECT currency_code FROM stock WHERE stock_id = $1`, req.StockID).Scan(&currencyCode); err != nil {
		return err
	}

	// Use the price sent from frontend (real-time price user saw when clicking sell)
	stockPrice := req.PricePerStock

	// Lock holding row to prevent oversell in concurrent requests.
	var ownedQty int
	if err = tx.QueryRowContext(ctx, `SELECT quantity FROM portfolio WHERE user_id = $1 AND stock_id = $2 FOR UPDATE`, req.UserID, req.StockID).Scan(&ownedQty); err != nil {
		if err == sql.ErrNoRows {
			return ErrInsufficientShares
		}
		return err
	}
	if ownedQty < req.Quantity {
		return ErrInsufficientShares
	}

	// Assumes one portfolio row per (user_id, stock_id).
	// If duplicates exist in DB, this update/delete logic needs redesign.
	now := time.Now()
	remainingQty := ownedQty - req.Quantity
	if remainingQty == 0 {
		_, err = tx.ExecContext(ctx, `DELETE FROM portfolio WHERE user_id = $1 AND stock_id = $2`, req.UserID, req.StockID)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE portfolio SET quantity = $1, transaction_time = $2 WHERE user_id = $3 AND stock_id = $4`, remainingQty, now, req.UserID, req.StockID)
	}
	if err != nil {
		return err
	}

	// Credit wallet only after shares are reduced.
	conversionRate := getConversionRate(currencyCode)
	sellAmount := stockPrice * float64(req.Quantity) * conversionRate

	// Log BEFORE balance
	var beforeBalance float64
	err = tx.QueryRowContext(ctx, `SELECT balance FROM wallet WHERE user_id = $1`, req.UserID).Scan(&beforeBalance)
	if err != nil {
		return fmt.Errorf("failed to read wallet balance: %w", err)
	}

	log.Printf("💰 [SELL] User: %s, Stock: %s, Qty: %d, Price: %.2f, Currency: %s, Rate: %.2f, Sell Amount: %.2f",
		req.UserID, req.StockID, req.Quantity, stockPrice, currencyCode, conversionRate, sellAmount)
	log.Printf("💵 [SELL] Balance BEFORE: %.2f", beforeBalance)

	walletRes, err := tx.ExecContext(ctx, `UPDATE wallet SET balance = balance + $1 WHERE user_id = $2`, sellAmount, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to update wallet: %w", err)
	}
	walletRows, err := walletRes.RowsAffected()
	if err != nil {
		return err
	}
	if walletRows == 0 {
		return fmt.Errorf("wallet not found for user: %s", req.UserID)
	}

	// Log AFTER balance
	var afterBalance float64
	err = tx.QueryRowContext(ctx, `SELECT balance FROM wallet WHERE user_id = $1`, req.UserID).Scan(&afterBalance)
	if err != nil {
		log.Printf("⚠️ [SELL] Failed to read balance after update: %v", err)
	}

	log.Printf("✅ [SELL] Wallet updated. BEFORE: %.2f, AFTER: %.2f, DIFFERENCE: %.2f", beforeBalance, afterBalance, afterBalance-beforeBalance)

	// Orders table is immutable trade history.
	_, err = tx.ExecContext(ctx, `INSERT INTO orders (stock_id, user_id, timestamp, status, quantity, price_per_stock)
		VALUES ($1, $2, $3, $4, $5, $6)`, req.StockID, req.UserID, now, "Filed", req.Quantity, stockPrice)
	if err != nil {
		return err
	}

	// Best-effort transaction confirmation notification.
	_, _ = tx.ExecContext(ctx, `
		INSERT INTO notifications (user_id, type, title, message)
		VALUES ($1, $2, $3, $4)`,
		req.UserID,
		"trade",
		"Sell order executed",
		"Your sell order was executed successfully.",
	)

	// Commit makes all sell changes visible together.
	return tx.Commit()
}
