package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"backend-go/auth"
	"backend-go/db"
	"backend-go/model"
)

// ── Stock Request/Response Structs ─────────────────────────────────────────────

type CreateStockRequest struct {
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type UpdateStockRequest struct {
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type StockResponse struct {
	StockID   string    `json:"stock_id"`
	Symbol    string    `json:"symbol"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Quantity  int       `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}

type WatchlistRequest struct {
	StockID string `json:"stock_id"`
}

type WatchlistResponse struct {
	WatchlistID string    `json:"watchlist_id"`
	UserID      string    `json:"user_id"`
	StockID     string    `json:"stock_id"`
	StockSymbol string    `json:"stock_symbol"`
	StockName   string    `json:"stock_name"`
	Price       float64   `json:"price"`
	Timestamp   time.Time `json:"timestamp"`
}

// ── Get All Stocks ────────────────────────────────────────────────────────────

func GetAllStocks(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	rows, err := db.Pool.Query(ctx,
		`SELECT stock_id, symbol, name, price, quantity, timestamp
		 FROM stock
		 ORDER BY name ASC`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch stocks")
		return
	}
	defer rows.Close()

	var stocks []StockResponse
	for rows.Next() {
		var stock StockResponse
		err := rows.Scan(
			&stock.StockID,
			&stock.Symbol,
			&stock.Name,
			&stock.Price,
			&stock.Quantity,
			&stock.Timestamp,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to parse stock data")
			return
		}
		stocks = append(stocks, stock)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stocks": stocks,
		"count":  len(stocks),
	})
}

// ── Get Stock by ID ────────────────────────────────────────────────────────────

func GetStockByID(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	stockID := r.PathValue("stock_id")

	if stockID == "" {
		writeError(w, http.StatusBadRequest, "stock_id is required")
		return
	}

	var stock StockResponse
	err := db.Pool.QueryRow(ctx,
		`SELECT stock_id, symbol, name, price, quantity, timestamp
		 FROM stock
		 WHERE stock_id = $1`,
		stockID,
	).Scan(
		&stock.StockID,
		&stock.Symbol,
		&stock.Name,
		&stock.Price,
		&stock.Quantity,
		&stock.Timestamp,
	)

	if err != nil {
		writeError(w, http.StatusNotFound, "stock not found")
		return
	}

	writeJSON(w, http.StatusOK, stock)
}

// ── Search Stocks ─────────────────────────────────────────────────────────────

func SearchStocks(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	query := r.URL.Query().Get("q")

	if query == "" {
		writeError(w, http.StatusBadRequest, "search query is required")
		return
	}

	searchPattern := "%" + query + "%"
	rows, err := db.Pool.Query(ctx,
		`SELECT stock_id, symbol, name, price, quantity, timestamp
		 FROM stock
		 WHERE name ILIKE $1 OR symbol ILIKE $1
		 ORDER BY name ASC
		 LIMIT 20`,
		searchPattern,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search stocks")
		return
	}
	defer rows.Close()

	var stocks []StockResponse
	for rows.Next() {
		var stock StockResponse
		err := rows.Scan(
			&stock.StockID,
			&stock.Symbol,
			&stock.Name,
			&stock.Price,
			&stock.Quantity,
			&stock.Timestamp,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to parse stock data")
			return
		}
		stocks = append(stocks, stock)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stocks": stocks,
		"count":  len(stocks),
		"query":  query,
	})
}

// ── Create Stock (Admin) ───────────────────────────────────────────────────────

func CreateStock(w http.ResponseWriter, r *http.Request) {
	// Verify user is authenticated (future: add admin role check)
	_, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req CreateStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Symbol == "" || req.Name == "" || req.Price <= 0 || req.Quantity < 0 {
		writeError(w, http.StatusBadRequest, "symbol, name, and price are required")
		return
	}

	ctx := context.Background()
	var stockID string
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO stock (symbol, name, price, quantity, timestamp)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING stock_id`,
		req.Symbol, req.Name, req.Price, req.Quantity, time.Now(),
	).Scan(&stockID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create stock")
		return
	}

	response := StockResponse{
		StockID:   stockID,
		Symbol:    req.Symbol,
		Name:      req.Name,
		Price:     req.Price,
		Quantity:  req.Quantity,
		Timestamp: time.Now(),
	}

	writeJSON(w, http.StatusCreated, response)
}

// ── Update Stock (Admin) ───────────────────────────────────────────────────────

func UpdateStock(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	stockID := r.PathValue("stock_id")
	if stockID == "" {
		writeError(w, http.StatusBadRequest, "stock_id is required")
		return
	}

	var req UpdateStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Price <= 0 || req.Quantity < 0 {
		writeError(w, http.StatusBadRequest, "price must be positive and quantity non-negative")
		return
	}

	ctx := context.Background()
	_, err := db.Pool.Exec(ctx,
		`UPDATE stock
		 SET price = $1, quantity = $2, timestamp = $3
		 WHERE stock_id = $4`,
		req.Price, req.Quantity, time.Now(), stockID,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update stock")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message":  "stock updated successfully",
		"stock_id": stockID,
	})
}

// ── Delete Stock (Admin) ───────────────────────────────────────────────────────

func DeleteStock(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	stockID := r.PathValue("stock_id")
	if stockID == "" {
		writeError(w, http.StatusBadRequest, "stock_id is required")
		return
	}

	ctx := context.Background()
	result, err := db.Pool.Exec(ctx,
		`DELETE FROM stock WHERE stock_id = $1`,
		stockID,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete stock")
		return
	}

	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "stock not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message":  "stock deleted successfully",
		"stock_id": stockID,
	})
}

// ── Add to Watchlist ───────────────────────────────────────────────────────────

func AddToWatchlist(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req WatchlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.StockID == "" {
		writeError(w, http.StatusBadRequest, "stock_id is required")
		return
	}

	ctx := context.Background()

	// Get stock details
	var stock model.Stock
	err := db.Pool.QueryRow(ctx,
		`SELECT stock_id, symbol, name, price, quantity, timestamp
		 FROM stock
		 WHERE stock_id = $1`,
		req.StockID,
	).Scan(
		&stock.StockID,
		&stock.Symbol,
		&stock.Name,
		&stock.Price,
		&stock.Quantity,
		&stock.Timestamp,
	)

	if err != nil {
		writeError(w, http.StatusNotFound, "stock not found")
		return
	}

	// Add to watchlist
	var watchlistID string
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO watchlist (user_id, watchlist_name, stock_id, quantity, price, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING watchlist_id`,
		userID, "default", req.StockID, stock.Quantity, stock.Price, time.Now(),
	).Scan(&watchlistID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add to watchlist")
		return
	}

	response := WatchlistResponse{
		WatchlistID: watchlistID,
		UserID:      userID,
		StockID:     stock.StockID,
		StockSymbol: stock.Symbol,
		StockName:   stock.Name,
		Price:       stock.Price,
		Timestamp:   time.Now(),
	}

	writeJSON(w, http.StatusCreated, response)
}

// ── Remove from Watchlist ──────────────────────────────────────────────────────

func RemoveFromWatchlist(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	watchlistID := r.PathValue("watchlist_id")
	if watchlistID == "" {
		writeError(w, http.StatusBadRequest, "watchlist_id is required")
		return
	}

	ctx := context.Background()
	result, err := db.Pool.Exec(ctx,
		`DELETE FROM watchlist
		 WHERE watchlist_id = $1 AND user_id = $2`,
		watchlistID, userID,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove from watchlist")
		return
	}

	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "watchlist item not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "removed from watchlist successfully",
	})
}

// ── Get User Watchlist ─────────────────────────────────────────────────────────

func GetWatchlist(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	ctx := context.Background()
	rows, err := db.Pool.Query(ctx,
		`SELECT w.watchlist_id, w.user_id, w.stock_id, s.symbol, s.name, s.price, w.timestamp
		 FROM watchlist w
		 JOIN stock s ON w.stock_id = s.stock_id
		 WHERE w.user_id = $1
		 ORDER BY w.timestamp DESC`,
		userID,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch watchlist")
		return
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var item model.WatchlistItem
		err := rows.Scan(
			&item.WatchlistID,
			&item.UserID,
			&item.StockID,
			&item.StockSymbol,
			&item.StockName,
			&item.Price,
			&item.Timestamp,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to parse watchlist data")
			return
		}
		items = append(items, map[string]interface{}{
			"watchlist_id": item.WatchlistID,
			"stock_id":     item.StockID,
			"stock_symbol": item.StockSymbol,
			"stock_name":   item.StockName,
			"price":        item.Price,
			"timestamp":    item.Timestamp,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"watchlist": items,
		"count":     len(items),
	})
}

// ── Get Stock Statistics ───────────────────────────────────────────────────────

func GetStockStats(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	stockID := r.PathValue("stock_id")

	if stockID == "" {
		writeError(w, http.StatusBadRequest, "stock_id is required")
		return
	}

	// Get stock info
	var stock StockResponse
	err := db.Pool.QueryRow(ctx,
		`SELECT stock_id, symbol, name, price, quantity, timestamp
		 FROM stock
		 WHERE stock_id = $1`,
		stockID,
	).Scan(
		&stock.StockID,
		&stock.Symbol,
		&stock.Name,
		&stock.Price,
		&stock.Quantity,
		&stock.Timestamp,
	)

	if err != nil {
		writeError(w, http.StatusNotFound, "stock not found")
		return
	}

	// Get trading volume
	var buyVolume, sellVolume int
	err = db.Pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN o.quantity > 0 THEN o.quantity ELSE 0 END), 0) as buy_volume,
			COALESCE(SUM(CASE WHEN o.quantity < 0 THEN o.quantity ELSE 0 END), 0) as sell_volume
		 FROM orders o
		 WHERE o.stock_id = $1 AND o.status = 'Filed'`,
		stockID,
	).Scan(&buyVolume, &sellVolume)

	if err != nil {
		// Continue even if stats fail
		buyVolume = 0
		sellVolume = 0
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stock":       stock,
		"buy_volume":  buyVolume,
		"sell_volume": -sellVolume,
	})
}

// ── Get Top Stocks ─────────────────────────────────────────────────────────────

func GetTopStocks(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	limitStr := r.URL.Query().Get("limit")
	limit := 10

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT stock_id, symbol, name, price, quantity, timestamp
		 FROM stock
		 ORDER BY timestamp DESC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch stocks")
		return
	}
	defer rows.Close()

	var stocks []StockResponse
	for rows.Next() {
		var stock StockResponse
		err := rows.Scan(
			&stock.StockID,
			&stock.Symbol,
			&stock.Name,
			&stock.Price,
			&stock.Quantity,
			&stock.Timestamp,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to parse stock data")
			return
		}
		stocks = append(stocks, stock)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stocks": stocks,
		"count":  len(stocks),
		"limit":  limit,
	})
}
