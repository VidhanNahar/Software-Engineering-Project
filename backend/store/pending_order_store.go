package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// PendingOrder represents a limit order waiting to be filled
type PendingOrder struct {
	OrderID        uuid.UUID  `json:"order_id"`
	UserID         uuid.UUID  `json:"user_id"`
	StockID        uuid.UUID  `json:"stock_id"`
	OrderType      string     `json:"order_type"` // BUY or SELL
	LimitPrice     float64    `json:"limit_price"`
	Quantity       int        `json:"quantity"`
	FilledQuantity int        `json:"filled_quantity"`
	Status         string     `json:"status"`        // PENDING, FILLED, PARTIALLY_FILLED, CANCELED
	TimeInForce    string     `json:"time_in_force"` // DAY, GTC
	CreatedAt      time.Time  `json:"created_at"`
	FilledAt       *time.Time `json:"filled_at,omitempty"`
	CanceledAt     *time.Time `json:"canceled_at,omitempty"`
}

// CreatePendingOrder creates a limit order that waits for price to reach limit
func (s *Store) CreatePendingOrder(ctx context.Context, userID, stockID uuid.UUID, orderType string, limitPrice float64, quantity int, timeInForce string) (uuid.UUID, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback()

	if orderType == "BUY" {
		var currencyCode string
		if err := tx.QueryRowContext(ctx, `SELECT currency_code FROM stock WHERE stock_id = $1`, stockID).Scan(&currencyCode); err != nil {
			return uuid.Nil, err
		}

		conversionRate := getConversionRate(currencyCode)
		reserveAmount := limitPrice * float64(quantity) * conversionRate

		var balance float64
		if err := tx.QueryRowContext(ctx, `SELECT balance FROM wallet WHERE user_id = $1 FOR UPDATE`, userID).Scan(&balance); err != nil {
			return uuid.Nil, err
		}
		if balance < reserveAmount {
			return uuid.Nil, fmt.Errorf("insufficient balance for pending order reservation")
		}

		if _, err := tx.ExecContext(ctx, `UPDATE wallet SET balance = balance - $1, locked_balance = locked_balance + $1 WHERE user_id = $2`, reserveAmount, userID); err != nil {
			return uuid.Nil, err
		}
	}

	var orderID uuid.UUID
	err = tx.QueryRowContext(ctx,
		`INSERT INTO pending_orders (user_id, stock_id, order_type, limit_price, quantity, time_in_force, status)
		 VALUES ($1, $2, $3, $4, $5, $6, 'PENDING')
		 RETURNING order_id`,
		userID, stockID, orderType, limitPrice, quantity, timeInForce).Scan(&orderID)
	if err != nil {
		return uuid.Nil, err
	}

	return orderID, tx.Commit()
}

// GetPendingOrdersForUser returns all pending orders for a user
func (s *Store) GetPendingOrdersForUser(ctx context.Context, userID uuid.UUID) ([]PendingOrder, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT order_id, user_id, stock_id, order_type, limit_price, quantity, filled_quantity, status, time_in_force, created_at, filled_at, canceled_at
		 FROM pending_orders
		 WHERE user_id = $1 AND status IN ('PENDING', 'PARTIALLY_FILLED')
		 ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []PendingOrder
	for rows.Next() {
		var order PendingOrder
		err := rows.Scan(&order.OrderID, &order.UserID, &order.StockID, &order.OrderType, &order.LimitPrice,
			&order.Quantity, &order.FilledQuantity, &order.Status, &order.TimeInForce, &order.CreatedAt, &order.FilledAt, &order.CanceledAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// GetLimitOrdersForUser returns all limit orders for a user, including filled and cancelled orders.
func (s *Store) GetLimitOrdersForUser(ctx context.Context, userID uuid.UUID) ([]PendingOrder, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT order_id, user_id, stock_id, order_type, limit_price, quantity, filled_quantity, status, time_in_force, created_at, filled_at, canceled_at
		 FROM pending_orders
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []PendingOrder
	for rows.Next() {
		var order PendingOrder
		err := rows.Scan(&order.OrderID, &order.UserID, &order.StockID, &order.OrderType, &order.LimitPrice,
			&order.Quantity, &order.FilledQuantity, &order.Status, &order.TimeInForce, &order.CreatedAt, &order.FilledAt, &order.CanceledAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// CancelPendingOrder cancels a pending order
func (s *Store) CancelPendingOrder(ctx context.Context, orderID uuid.UUID) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID, stockID uuid.UUID
	var orderType string
	var limitPrice float64
	var quantity, filledQty int
	if err := tx.QueryRowContext(ctx, `SELECT user_id, stock_id, order_type, limit_price, quantity, filled_quantity FROM pending_orders WHERE order_id = $1 AND status IN ('PENDING', 'PARTIALLY_FILLED') FOR UPDATE`, orderID).Scan(&userID, &stockID, &orderType, &limitPrice, &quantity, &filledQty); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("order not found or already filled")
		}
		return err
	}

	if orderType == "BUY" {
		var currencyCode string
		if err := tx.QueryRowContext(ctx, `SELECT currency_code FROM stock WHERE stock_id = $1`, stockID).Scan(&currencyCode); err != nil {
			return err
		}
		conversionRate := getConversionRate(currencyCode)
		remainingQty := quantity - filledQty
		refundAmount := limitPrice * float64(remainingQty) * conversionRate
		if _, err := tx.ExecContext(ctx, `UPDATE wallet SET balance = balance + $1, locked_balance = GREATEST(locked_balance - $1, 0) WHERE user_id = $2`, refundAmount, userID); err != nil {
			return err
		}
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE pending_orders SET status = 'CANCELED', canceled_at = CURRENT_TIMESTAMP WHERE order_id = $1`,
		orderID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// MatchPendingOrders checks all pending buy/sell orders and executes those where price is favorable
// This should be called when market price updates
func (s *Store) MatchPendingOrders(ctx context.Context, stockID uuid.UUID, currentPrice float64) error {
	// Get all pending buy orders where current price <= limit price (favorable for buyer)
	buyOrders, err := s.db.QueryContext(ctx,
		`SELECT order_id, user_id, stock_id, limit_price, quantity, filled_quantity
		 FROM pending_orders
		 WHERE stock_id = $1 AND order_type = 'BUY' AND status = 'PENDING' AND limit_price >= $2
		 ORDER BY limit_price DESC, created_at ASC`,
		stockID, currentPrice)
	if err != nil {
		return err
	}
	defer buyOrders.Close()

	for buyOrders.Next() {
		var orderID, userID, sID uuid.UUID
		var limitPrice float64
		var quantity, filledQty int

		err := buyOrders.Scan(&orderID, &userID, &sID, &limitPrice, &quantity, &filledQty)
		if err != nil {
			log.Printf("❌ Error scanning buy order: %v", err)
			continue
		}

		remainingQty := quantity - filledQty
		if remainingQty > 0 {
			// Execute at current market price (buyer benefits from better prices)
			executionPrice := currentPrice

			err := s.executePendingBuyOrder(ctx, orderID, userID, sID, limitPrice, executionPrice, remainingQty)
			if err != nil {
				log.Printf("❌ Error executing pending buy order %s: %v", orderID, err)
				continue
			}
			log.Printf("✅ [PENDING BUY MATCH] Order %s executed at ₹%.2f", orderID, executionPrice)
		}
	}

	// Get all pending sell orders where current price >= limit price (favorable for seller)
	sellOrders, err := s.db.QueryContext(ctx,
		`SELECT order_id, user_id, stock_id, limit_price, quantity, filled_quantity
		 FROM pending_orders
		 WHERE stock_id = $1 AND order_type = 'SELL' AND status = 'PENDING' AND limit_price <= $2
		 ORDER BY limit_price ASC, created_at ASC`,
		stockID, currentPrice)
	if err != nil {
		return err
	}
	defer sellOrders.Close()

	for sellOrders.Next() {
		var orderID, userID, sID uuid.UUID
		var limitPrice float64
		var quantity, filledQty int

		err := sellOrders.Scan(&orderID, &userID, &sID, &limitPrice, &quantity, &filledQty)
		if err != nil {
			log.Printf("❌ Error scanning sell order: %v", err)
			continue
		}

		remainingQty := quantity - filledQty
		if remainingQty > 0 {
			// Execute at current market price (seller benefits from higher prices)
			executionPrice := currentPrice

			err := s.executePendingSellOrder(ctx, orderID, userID, sID, executionPrice, remainingQty)
			if err != nil {
				log.Printf("❌ Error executing pending sell order %s: %v", orderID, err)
				continue
			}
			log.Printf("✅ [PENDING SELL MATCH] Order %s executed at ₹%.2f", orderID, executionPrice)
		}
	}

	return nil
}

// executePendingBuyOrder executes a pending buy order
func (s *Store) executePendingBuyOrder(ctx context.Context, orderID, userID, stockID uuid.UUID, limitPrice, executionPrice float64, quantity int) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get currency and calculate reserved amount and final cost
	var currencyCode string
	err = tx.QueryRowContext(ctx, `SELECT currency_code FROM stock WHERE stock_id = $1`, stockID).Scan(&currencyCode)
	if err != nil {
		return err
	}

	conversionRate := getConversionRate(currencyCode)
	reservedAmount := limitPrice * float64(quantity) * conversionRate
	executionCost := executionPrice * float64(quantity) * conversionRate
	refundAmount := reservedAmount - executionCost

	if refundAmount < 0 {
		refundAmount = 0
	}

	// Move reserved funds back to available balance, then deduct final execution cost from balance reserve.
	_, err = tx.ExecContext(ctx, `UPDATE wallet SET locked_balance = GREATEST(locked_balance - $1, 0), balance = balance + $2 WHERE user_id = $3`, reservedAmount, refundAmount, userID)
	if err != nil {
		return err
	}

	// Update portfolio
	now := time.Now()
	var existingQty int
	var existingPrice float64

	err = tx.QueryRowContext(ctx, `SELECT quantity, price FROM portfolio WHERE user_id = $1 AND stock_id = $2`,
		userID, stockID).Scan(&existingQty, &existingPrice)
	if err == nil {
		// Stock already exists - calculate new average price
		totalInvestedValue := existingPrice*float64(existingQty) + executionPrice*float64(quantity)
		newQuantity := existingQty + quantity
		avgPrice := totalInvestedValue / float64(newQuantity)

		_, err = tx.ExecContext(ctx, `UPDATE portfolio SET quantity = quantity + $1, price = $2, transaction_time = $3
									WHERE user_id = $4 AND stock_id = $5`,
			quantity, avgPrice, now, userID, stockID)
		if err != nil {
			return err
		}
	} else if err == sql.ErrNoRows {
		// First buy for this stock
		_, err = tx.ExecContext(ctx, `INSERT INTO portfolio (user_id, stock_id, price, quantity, transaction_time)
									VALUES ($1, $2, $3, $4, $5)`,
			userID, stockID, executionPrice, quantity, now)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	// Update pending order status
	_, err = tx.ExecContext(ctx, `UPDATE pending_orders 
							SET status = 'FILLED', filled_quantity = quantity, filled_at = CURRENT_TIMESTAMP
							WHERE order_id = $1`, orderID)
	if err != nil {
		return err
	}

	// Insert order history
	_, err = tx.ExecContext(ctx, `INSERT INTO orders (stock_id, user_id, timestamp, status, quantity, price_per_stock)
								VALUES ($1, $2, $3, $4, $5, $6)`,
		stockID, userID, now, "Filled", quantity, executionPrice)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// executePendingSellOrder executes a pending sell order
func (s *Store) executePendingSellOrder(ctx context.Context, orderID, userID, stockID uuid.UUID, executionPrice float64, quantity int) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if user owns enough shares
	var ownedQty int
	var ownedPrice float64
	err = tx.QueryRowContext(ctx, `SELECT quantity, price FROM portfolio WHERE user_id = $1 AND stock_id = $2 FOR UPDATE`,
		userID, stockID).Scan(&ownedQty, &ownedPrice)
	if err == sql.ErrNoRows {
		return errors.New("user does not own this stock")
	}
	if err != nil {
		return err
	}
	if ownedQty < quantity {
		return fmt.Errorf("insufficient shares: own %d, trying to sell %d", ownedQty, quantity)
	}

	// Get currency and calculate proceeds
	var currencyCode string
	err = tx.QueryRowContext(ctx, `SELECT currency_code FROM stock WHERE stock_id = $1`, stockID).Scan(&currencyCode)
	if err != nil {
		return err
	}

	conversionRate := getConversionRate(currencyCode)
	proceeds := executionPrice * float64(quantity) * conversionRate

	// Credit wallet
	_, err = tx.ExecContext(ctx, `UPDATE wallet SET balance = balance + $1 WHERE user_id = $2`, proceeds, userID)
	if err != nil {
		return err
	}

	// Update portfolio
	now := time.Now()
	newQty := ownedQty - quantity

	if newQty > 0 {
		_, err = tx.ExecContext(ctx, `UPDATE portfolio SET quantity = $1, transaction_time = $2 WHERE user_id = $3 AND stock_id = $4`,
			newQty, now, userID, stockID)
		if err != nil {
			return err
		}
	} else {
		// Sold all shares - remove from portfolio
		_, err = tx.ExecContext(ctx, `DELETE FROM portfolio WHERE user_id = $1 AND stock_id = $2`, userID, stockID)
		if err != nil {
			return err
		}
	}

	// Update pending order status
	_, err = tx.ExecContext(ctx, `UPDATE pending_orders 
							SET status = 'FILLED', filled_quantity = quantity, filled_at = CURRENT_TIMESTAMP
							WHERE order_id = $1`, orderID)
	if err != nil {
		return err
	}

	// Insert order history
	_, err = tx.ExecContext(ctx, `INSERT INTO orders (stock_id, user_id, timestamp, status, quantity, price_per_stock)
								VALUES ($1, $2, $3, $4, $5, $6)`,
		stockID, userID, now, "Filled", quantity, executionPrice)
	if err != nil {
		return err
	}

	return tx.Commit()
}
