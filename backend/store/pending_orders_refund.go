package store

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// ReleasePendingOrdersOnMarketClose refunds all DAY pending orders when market closes
// GTC (Good Till Cancelled) orders persist across market sessions
// For BUY orders: Returns the amount to wallet
// For SELL orders: Returns shares to portfolio
func (s *Store) ReleasePendingOrdersOnMarketClose(ctx context.Context) error {
	// Get all pending DAY orders (GTC orders persist)
	rows, err := s.db.QueryContext(ctx,
		`SELECT order_id, user_id, stock_id, order_type, limit_price, quantity, filled_quantity
		 FROM pending_orders
		 WHERE status = 'PENDING' AND time_in_force = 'DAY'
		 ORDER BY created_at ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var orders []struct {
		OrderID        string
		UserID         string
		StockID        string
		OrderType      string
		LimitPrice     float64
		Quantity       int
		FilledQuantity int
	}

	for rows.Next() {
		var order struct {
			OrderID        string
			UserID         string
			StockID        string
			OrderType      string
			LimitPrice     float64
			Quantity       int
			FilledQuantity int
		}

		err := rows.Scan(&order.OrderID, &order.UserID, &order.StockID, &order.OrderType,
			&order.LimitPrice, &order.Quantity, &order.FilledQuantity)
		if err != nil {
			log.Printf("❌ Error scanning pending order: %v", err)
			continue
		}
		orders = append(orders, order)
	}

	// Process each pending order
	for _, order := range orders {
		if order.OrderType == "BUY" {
			// Refund the amount to wallet
			err := s.refundBuyOrder(ctx, order.OrderID, order.UserID, order.StockID, order.LimitPrice, order.Quantity, order.FilledQuantity)
			if err != nil {
				log.Printf("❌ Error refunding buy order %s: %v", order.OrderID, err)
				continue
			}
			log.Printf("✅ [REFUND BUY] Order %s refunded to wallet", order.OrderID)
		} else if order.OrderType == "SELL" {
			// Return shares to portfolio
			err := s.refundSellOrder(ctx, order.OrderID, order.UserID, order.StockID, order.Quantity, order.FilledQuantity)
			if err != nil {
				log.Printf("❌ Error refunding sell order %s: %v", order.OrderID, err)
				continue
			}
			log.Printf("✅ [REFUND SELL] Order %s shares returned to portfolio", order.OrderID)
		}
	}

	return rows.Err()
}

// refundBuyOrder refunds a pending buy order back to wallet
func (s *Store) refundBuyOrder(ctx context.Context, orderID, userID, stockID string, limitPrice float64, quantity, filledQty int) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get currency for conversion
	var currencyCode string
	err = tx.QueryRowContext(ctx, `SELECT currency_code FROM stock WHERE stock_id = $1`, stockID).Scan(&currencyCode)
	if err != nil {
		return err
	}

	conversionRate := getConversionRate(currencyCode)
	remainingQty := quantity - filledQty
	refundAmount := limitPrice * float64(remainingQty) * conversionRate

	// Refund to wallet
	_, err = tx.ExecContext(ctx, `UPDATE wallet SET balance = balance + $1 WHERE user_id = $2`, refundAmount, userID)
	if err != nil {
		return err
	}

	// Mark order as CANCELED
	_, err = tx.ExecContext(ctx,
		`UPDATE pending_orders SET status = 'CANCELED', canceled_at = CURRENT_TIMESTAMP WHERE order_id = $1`,
		orderID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// refundSellOrder refunds a pending sell order by returning shares to portfolio
func (s *Store) refundSellOrder(ctx context.Context, orderID, userID, stockID string, quantity, filledQty int) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	remainingQty := quantity - filledQty

	// Get current portfolio entry
	var existingQty int
	var existingPrice float64
	err = tx.QueryRowContext(ctx,
		`SELECT quantity, price FROM portfolio WHERE user_id = $1 AND stock_id = $2`,
		userID, stockID).Scan(&existingQty, &existingPrice)

	now := time.Now()

	if err == sql.ErrNoRows {
		// Portfolio entry doesn't exist - create it with the refunded shares
		// Use average price of 0 since this is a refund
		_, err = tx.ExecContext(ctx,
			`INSERT INTO portfolio (user_id, stock_id, quantity, price, transaction_time)
			 VALUES ($1, $2, $3, $4, $5)`,
			userID, stockID, remainingQty, 0, now)
		if err != nil {
			return err
		}
	} else if err == nil {
		// Portfolio entry exists - add back the shares
		newQty := existingQty + remainingQty
		_, err = tx.ExecContext(ctx,
			`UPDATE portfolio SET quantity = $1, transaction_time = $2 WHERE user_id = $3 AND stock_id = $4`,
			newQty, now, userID, stockID)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	// Mark order as CANCELED
	_, err = tx.ExecContext(ctx,
		`UPDATE pending_orders SET status = 'CANCELED', canceled_at = CURRENT_TIMESTAMP WHERE order_id = $1`,
		orderID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
