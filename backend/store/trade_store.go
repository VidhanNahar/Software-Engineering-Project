package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrInsufficientShares  = errors.New("insufficient shares in portfolio")
)

type TradeRequest struct {
	UserID   uuid.UUID
	StockID  uuid.UUID
	Quantity int
}

func (s *Store) ExecuteBuyTx(ctx context.Context, req TradeRequest) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// TODO: Step-by-step add buy checks + updates here.

	if req.Quantity <= 0 {
		return errors.New("Quantity must be greater than 0")
	}

	var stockPrice float64
	err = tx.QueryRowContext(ctx, `SELECT price FROM stock WHERE stock_id = $1`, req.StockID).Scan(&stockPrice)
	if err != nil {
		return err
	}

	totalCost := stockPrice * float64(req.Quantity)
	// check balance >= totalcost
	var balance float64
	err = tx.QueryRowContext(ctx, `SELECT balance from wallet WHERE user_id=$1 FOR UPDATE`, req.UserID).Scan(&balance)
	if err != nil {
		return err
	}
	if balance < totalCost {
		return ErrInsufficientBalance
	}

	//Update balance
	_, err = tx.ExecContext(ctx, `UPDATE wallet SET balance = balance - $1 WHERE user_id=$2`, totalCost, req.UserID)
	if err != nil {
		return err
	}

	//Wallet deduction done

	// Now updating portfolio holdings

	now := time.Now()
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
		_, err = tx.ExecContext(ctx, `INSERT INTO portfolio (user_id, stock_id, transaction_time, price, quantity)
  								VALUES ($1, $2, $3, $4, $5)`, req.UserID, req.StockID, now, stockPrice, req.Quantity)
		if err != nil {
			return err
		}
	}

	/////
	return tx.Commit()
}

func (s *Store) ExecuteSellTx(ctx context.Context, req TradeRequest) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// TODO: Step-by-step add sell checks + updates here.
	_ = time.Now()

	return tx.Commit()
}
