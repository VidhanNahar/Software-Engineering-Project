package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

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
	UserID   uuid.UUID
	StockID  uuid.UUID
	Quantity int
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

	// Price is fetched inside tx so all reads/writes share one boundary.
	var stockPrice float64
	if err = tx.QueryRowContext(ctx, `SELECT price FROM stock WHERE stock_id = $1`, req.StockID).Scan(&stockPrice); err != nil {
		return err
	}

	totalCost := stockPrice * float64(req.Quantity)
	// Lock wallet row to prevent double-spend in concurrent buys.
	var balance float64
	err = tx.QueryRowContext(ctx, `SELECT balance from wallet WHERE user_id=$1 FOR UPDATE`, req.UserID).Scan(&balance)
	if err != nil {
		return err
	}
	if balance < totalCost {
		return ErrInsufficientBalance
	}

	// Debit wallet only after passing balance validation.
	walletRes, err := tx.ExecContext(ctx, `UPDATE wallet SET balance = balance - $1 WHERE user_id=$2`, totalCost, req.UserID)
	if err != nil {
		return err
	}
	walletRows, err := walletRes.RowsAffected()
	if err != nil {
		return err
	}
	if walletRows == 0 {
		return ErrWalletNotFound
	}

	now := time.Now()
	// Assumes one portfolio row per (user_id, stock_id).
	// If duplicates exist in DB, this UPDATE may affect multiple rows.
	res, err := tx.ExecContext(ctx, `UPDATE portfolio
								SET quantity = quantity + $1, transaction_time = $2, price = $3
								WHERE user_id = $4 AND stock_id = $5`,
		req.Quantity, now, stockPrice, req.UserID, req.StockID)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		// First buy for this stock creates the portfolio position.
		_, err = tx.ExecContext(ctx, `INSERT INTO portfolio (user_id, stock_id, transaction_time, price, quantity)
  								VALUES ($1, $2, $3, $4, $5)`, req.UserID, req.StockID, now, stockPrice, req.Quantity)
		if err != nil {
			return err
		}
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

	// Current price is used to compute credited sell amount.
	var stockPrice float64
	if err = tx.QueryRowContext(ctx, `SELECT price FROM stock WHERE stock_id = $1`, req.StockID).Scan(&stockPrice); err != nil {
		return err
	}

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
	sellAmount := stockPrice * float64(req.Quantity)
	walletRes, err := tx.ExecContext(ctx, `UPDATE wallet SET balance = balance + $1 WHERE user_id = $2`, sellAmount, req.UserID)
	if err != nil {
		return err
	}
	walletRows, err := walletRes.RowsAffected()
	if err != nil {
		return err
	}
	if walletRows == 0 {
		return ErrWalletNotFound
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
		"Sell order executed",
		"Your sell order was executed successfully.",
	)

	// Commit makes all sell changes visible together.
	return tx.Commit()
}
