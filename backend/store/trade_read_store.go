package store

import (
	"backend-go/model"

	"github.com/google/uuid"
)

// GetStocks returns current market quotes for all stocks.
func (s *Store) GetStocks() ([]model.StockQuote, error) {
	rows, err := s.db.Query(`SELECT stock_id, name, price, timestamp, quantity FROM stock ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotes := make([]model.StockQuote, 0)
	for rows.Next() {
		var quote model.StockQuote
		if err := rows.Scan(&quote.StockID, &quote.Name, &quote.Price, &quote.Timestamp, &quote.AvailableQuantity); err != nil {
			return nil, err
		}
		quotes = append(quotes, quote)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return quotes, nil
}

// GetPortfolioByUser returns user holdings with current valuation.
func (s *Store) GetPortfolioByUser(userID uuid.UUID) ([]model.PortfolioPosition, error) {
	rows, err := s.db.Query(`
		SELECT p.user_id, p.stock_id, st.name, p.quantity, p.price, st.price, (p.quantity * st.price), p.transaction_time
		FROM portfolio p
		INNER JOIN stock st ON st.stock_id = p.stock_id
		WHERE p.user_id = $1
		ORDER BY p.transaction_time DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make([]model.PortfolioPosition, 0)
	for rows.Next() {
		var position model.PortfolioPosition
		if err := rows.Scan(
			&position.UserID,
			&position.StockID,
			&position.StockName,
			&position.Quantity,
			&position.AvgBuyPrice,
			&position.CurrentPrice,
			&position.PositionValue,
			&position.LastUpdateTime,
		); err != nil {
			return nil, err
		}
		positions = append(positions, position)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return positions, nil
}

// GetOrdersByUser returns transaction history in reverse chronology.
func (s *Store) GetOrdersByUser(userID uuid.UUID) ([]model.OrderRecord, error) {
	rows, err := s.db.Query(`
		SELECT order_id, stock_id, user_id, timestamp, status, quantity, price_per_stock
		FROM orders
		WHERE user_id = $1
		ORDER BY timestamp DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.OrderRecord, 0)
	for rows.Next() {
		var order model.OrderRecord
		if err := rows.Scan(&order.OrderID, &order.StockID, &order.UserID, &order.Timestamp, &order.Status, &order.Quantity, &order.PricePerStock); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}
