package controller

import (
	"backend-go/market"
	"backend-go/middleware"
	"backend-go/model"
	"backend-go/store"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TradeHandler serves trade-related APIs.
type TradeHandler struct {
	store         *store.Store
	marketService *market.MarketService
	broadcaster   *market.WebSocketBroadcaster
}

func NewTradeHandler(s *store.Store, marketService *market.MarketService, broadcaster *market.WebSocketBroadcaster) *TradeHandler {
	return &TradeHandler{store: s, marketService: marketService, broadcaster: broadcaster}
}

func (h *TradeHandler) getRequester(r *http.Request) (*model.User, error) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, errors.New("unauthorized")
	}
	return h.store.GetUserByID(userID)
}

func (h *TradeHandler) requireTradableUser(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	if !user.IsKYCVerified || user.Role == "guest" {
		http.Error(w, "Complete KYC to start trading", http.StatusForbidden)
		return nil, false
	}
	return user, true
}

func (h *TradeHandler) requireAdmin(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	if user.Role != "admin" {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return nil, false
	}
	return user, true
}

// BuyStock executes a buy transaction or creates pending limit order for authenticated user.
func (h *TradeHandler) BuyStock(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireTradableUser(w, r)
	if !ok {
		return
	}

	// Check if market is open
	isOpen, err := h.store.IsMarketOpen()
	if err != nil {
		http.Error(w, "Failed to check market status", http.StatusInternalServerError)
		return
	}
	if !isOpen {
		http.Error(w, "Market is closed. Trading is not allowed at this time.", http.StatusForbidden)
		return
	}

	var req model.TradeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate order type
	if req.OrderType == "" {
		req.OrderType = "MARKET" // Default to market order
	}
	if req.OrderType != "MARKET" && req.OrderType != "LIMIT" {
		http.Error(w, "Invalid order type. Must be MARKET or LIMIT", http.StatusBadRequest)
		return
	}

	// Validate limit price for limit orders
	if req.OrderType == "LIMIT" && req.PricePerStock < 0 {
		http.Error(w, "Limit price must be greater than or equal to 0", http.StatusBadRequest)
		return
	}

	stock, err := h.store.GetStockByID(req.StockID)
	if err != nil {
		http.Error(w, "Stock not found", http.StatusNotFound)
		return
	}

	currentPrice := stock.Price
	executionPrice := currentPrice

	// For limit orders, execute immediately when the current price is already
	// favorable; otherwise create a pending order that will be filled later.
	if req.OrderType == "LIMIT" {
		if currentPrice > req.PricePerStock {
			if req.TimeInForce == "" {
				req.TimeInForce = "DAY"
			}

			orderID, err := h.store.CreatePendingOrder(r.Context(), user.UserID, req.StockID, "BUY", req.PricePerStock, req.Quantity, req.TimeInForce)
			if err != nil {
				http.Error(w, "Failed to create pending order", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"message":     "Buy limit order created and pending",
				"order_id":    orderID,
				"stock_id":    req.StockID,
				"quantity":    req.Quantity,
				"limit_price": req.PricePerStock,
				"status":      "PENDING",
				"note":        fmt.Sprintf("Order will execute when market price is ₹%.2f or lower", req.PricePerStock),
			})
			return
		}
		// Price is already favorable, fall through to immediate execution at current price.
	}

	err = h.store.ExecuteBuyTx(r.Context(), store.TradeRequest{UserID: user.UserID, StockID: req.StockID, Quantity: req.Quantity, PricePerStock: executionPrice})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrInsufficientBalance):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, store.ErrWalletNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "Failed to execute buy trade", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":         "Buy order executed",
		"stock_id":        req.StockID,
		"quantity":        req.Quantity,
		"executed_price":  executionPrice,
		"requested_price": req.PricePerStock,
	})
}

// SellStock executes a sell transaction or creates pending limit order for authenticated user.
func (h *TradeHandler) SellStock(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireTradableUser(w, r)
	if !ok {
		return
	}

	// Check if market is open
	isOpen, err := h.store.IsMarketOpen()
	if err != nil {
		http.Error(w, "Failed to check market status", http.StatusInternalServerError)
		return
	}
	if !isOpen {
		http.Error(w, "Market is closed. Trading is not allowed at this time.", http.StatusForbidden)
		return
	}

	var req model.TradeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate order type
	if req.OrderType == "" {
		req.OrderType = "MARKET" // Default to market order
	}
	if req.OrderType != "MARKET" && req.OrderType != "LIMIT" {
		http.Error(w, "Invalid order type. Must be MARKET or LIMIT", http.StatusBadRequest)
		return
	}

	// Validate limit price for limit orders
	if req.OrderType == "LIMIT" && req.PricePerStock < 0 {
		http.Error(w, "Limit price must be greater than or equal to 0", http.StatusBadRequest)
		return
	}

	stock, err := h.store.GetStockByID(req.StockID)
	if err != nil {
		http.Error(w, "Stock not found", http.StatusNotFound)
		return
	}

	currentPrice := stock.Price
	executionPrice := currentPrice

	positions, err := h.store.GetPortfolioByUser(user.UserID)
	if err != nil {
		http.Error(w, "Failed to verify available holdings", http.StatusInternalServerError)
		return
	}

	availableQty := 0
	for _, p := range positions {
		if p.StockID == req.StockID {
			availableQty = p.AvailableQty
			break
		}
	}

	if req.Quantity > availableQty {
		http.Error(w, fmt.Sprintf("Only %d shares are available to sell. Remaining shares are locked in pending sell limit orders.", availableQty), http.StatusConflict)
		return
	}

	// For limit orders, execute immediately when the current price is already
	// favorable; otherwise create a pending order that will be filled later.
	if req.OrderType == "LIMIT" {
		if currentPrice < req.PricePerStock {
			if req.TimeInForce == "" {
				req.TimeInForce = "DAY"
			}

			orderID, err := h.store.CreatePendingOrder(r.Context(), user.UserID, req.StockID, "SELL", req.PricePerStock, req.Quantity, req.TimeInForce)
			if err != nil {
				http.Error(w, "Failed to create pending order", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"message":     "Sell limit order created and pending",
				"order_id":    orderID,
				"stock_id":    req.StockID,
				"quantity":    req.Quantity,
				"limit_price": req.PricePerStock,
				"status":      "PENDING",
				"note":        fmt.Sprintf("Order will execute when market price is ₹%.2f or higher", req.PricePerStock),
			})
			return
		}
		// Price is already favorable, fall through to immediate execution at current price.
	}

	err = h.store.ExecuteSellTx(r.Context(), store.TradeRequest{UserID: user.UserID, StockID: req.StockID, Quantity: req.Quantity, PricePerStock: executionPrice})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrInsufficientShares):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, store.ErrWalletNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "Failed to execute sell trade", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message":         "Sell order executed",
		"stock_id":        req.StockID,
		"quantity":        req.Quantity,
		"executed_price":  executionPrice,
		"requested_price": req.PricePerStock,
	})
}

// GetStocks returns all stock quotes.
func (h *TradeHandler) GetStocks(w http.ResponseWriter, r *http.Request) {
	isOpen, err := h.store.IsMarketOpen()
	if err != nil {
		http.Error(w, "Failed to check market status", http.StatusInternalServerError)
		return
	}

	stocks, err := h.store.GetStocks()
	if err != nil {
		http.Error(w, "Failed to fetch stocks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"stocks":      stocks,
		"count":       len(stocks),
		"market_open": isOpen,
	})
}

// GetPortfolio returns authenticated user's holdings.
func (h *TradeHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	positions, err := h.store.GetPortfolioByUser(user.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch portfolio", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Also fetch pending orders to show alongside holdings
	pendingOrders, err := h.store.GetPendingOrdersForUser(r.Context(), user.UserID)
	if err != nil {
		// Don't fail the response, just skip pending orders
		pendingOrders = []store.PendingOrder{}
	}

	json.NewEncoder(w).Encode(map[string]any{
		"holdings":       positions,
		"count":          len(positions),
		"pending_orders": pendingOrders,
		"pending_count":  len(pendingOrders),
	})
}

// GetOrders returns authenticated user's transaction history.
func (h *TradeHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.store.GetOrdersByUser(user.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"orders": orders, "count": len(orders)})
}

func (h *TradeHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	balance, locked, err := h.store.GetWalletByUser(user.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch wallet", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"balance":        balance,
		"locked_balance": locked,
	})
}

// GetPendingOrders returns all pending limit orders for authenticated user.
func (h *TradeHandler) GetPendingOrders(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	activeOrders, err := h.store.GetPendingOrdersForUser(r.Context(), user.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch pending orders", http.StatusInternalServerError)
		return
	}

	limitOrders, err := h.store.GetLimitOrdersForUser(r.Context(), user.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch limit orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"pending_orders": activeOrders,
		"limit_orders":   limitOrders,
		"count":          len(limitOrders),
		"pending_count":  len(activeOrders),
	})
}

// CancelPendingOrder cancels a pending limit order
func (h *TradeHandler) CancelPendingOrder(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isOpen, err := h.store.IsMarketOpen()
	if err != nil {
		http.Error(w, "Failed to check market status", http.StatusInternalServerError)
		return
	}
	if !isOpen {
		http.Error(w, "Market is closed. Pending orders cannot be canceled right now.", http.StatusForbidden)
		return
	}

	orderID := mux.Vars(r)["order_id"]
	if orderID == "" {
		http.Error(w, "Missing order_id parameter", http.StatusBadRequest)
		return
	}

	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		http.Error(w, "Invalid order_id format", http.StatusBadRequest)
		return
	}

	// Verify the order belongs to the user
	orders, err := h.store.GetPendingOrdersForUser(r.Context(), user.UserID)
	if err != nil {
		http.Error(w, "Failed to verify order", http.StatusInternalServerError)
		return
	}

	orderFound := false
	for _, order := range orders {
		if order.OrderID == orderUUID {
			orderFound = true
			break
		}
	}

	if !orderFound {
		http.Error(w, "Order not found or already filled", http.StatusNotFound)
		return
	}

	// Cancel the order
	err = h.store.CancelPendingOrder(r.Context(), orderUUID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message":  "Pending order canceled successfully",
		"order_id": orderID,
	})
}

// ReleasePendingOrdersOnMarketClose refunds all pending orders when market closes
// This is called by admin when stopping the market
func (h *TradeHandler) ReleasePendingOrdersOnMarketClose(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	err := h.store.ReleasePendingOrdersOnMarketClose(r.Context())
	if err != nil {
		log.Printf("❌ Error releasing pending orders: %v", err)
		http.Error(w, "Failed to release pending orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "All pending orders refunded successfully",
		"admin":   user.UserName,
		"time":    time.Now(),
	})
}

func (h *TradeHandler) SearchStocks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode(map[string]any{"stocks": []any{}, "count": 0})
		return
	}

	stocks, err := h.store.SearchStocks(query)
	if err != nil {
		http.Error(w, "Failed to search stocks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"stocks": stocks, "count": len(stocks)})
}

func (h *TradeHandler) GetStockByID(w http.ResponseWriter, r *http.Request) {
	stockID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid stock id", http.StatusBadRequest)
		return
	}

	stock, err := h.store.GetStockByID(stockID)
	if err != nil {
		if errors.Is(err, store.ErrStockNotFound) {
			http.Error(w, "Stock not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch stock", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stock)
}

func (h *TradeHandler) GetStockTicksBySymbol(w http.ResponseWriter, r *http.Request) {
	symbol := mux.Vars(r)["symbol"]
	if symbol == "" {
		http.Error(w, "Invalid symbol", http.StatusBadRequest)
		return
	}

	limit := 200
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err == nil && parsed > 0 {
			limit = parsed
		}
	}

	ticks, err := h.store.GetStockTicksBySymbol(symbol, limit)
	if err != nil {
		http.Error(w, "Failed to fetch ticks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"symbol": symbol,
		"ticks":  ticks,
		"count":  len(ticks),
	})
}

func (h *TradeHandler) GetStockCandlesBySymbol(w http.ResponseWriter, r *http.Request) {
	symbol := mux.Vars(r)["symbol"]
	if symbol == "" {
		http.Error(w, "Invalid symbol", http.StatusBadRequest)
		return
	}

	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "1m"
	}

	limit := 200
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err == nil && parsed > 0 {
			limit = parsed
		}
	}

	candles, err := h.store.GetStockCandlesBySymbol(symbol, timeframe, limit)
	if err != nil {
		http.Error(w, "Failed to fetch candles", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"symbol":    symbol,
		"timeframe": timeframe,
		"candles":   candles,
		"count":     len(candles),
	})
}

func (h *TradeHandler) GetStockBySymbol(w http.ResponseWriter, r *http.Request) {
	symbol := mux.Vars(r)["symbol"]
	if symbol == "" {
		http.Error(w, "Invalid symbol", http.StatusBadRequest)
		return
	}

	stock, err := h.store.GetStockBySymbol(symbol)
	if err != nil {
		if errors.Is(err, store.ErrStockNotFound) {
			http.Error(w, "Stock not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch stock", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stock)
}

func (h *TradeHandler) GetStockHistory(w http.ResponseWriter, r *http.Request) {
	stockID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid stock id", http.StatusBadRequest)
		return
	}

	limit := 120
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := h.store.GetStockHistory(stockID, limit)
	if err != nil {
		http.Error(w, "Failed to fetch stock history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"history": history, "count": len(history)})
}

func (h *TradeHandler) GetStockStats(w http.ResponseWriter, r *http.Request) {
	stockID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid stock id", http.StatusBadRequest)
		return
	}

	stock, err := h.store.GetStockByID(stockID)
	if err != nil {
		if errors.Is(err, store.ErrStockNotFound) {
			http.Error(w, "Stock not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch stock", http.StatusInternalServerError)
		return
	}

	history, err := h.store.GetStockHistory(stockID, 30)
	if err != nil {
		http.Error(w, "Failed to fetch stock history", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"stock":   stock,
		"history": history,
	})
}

func (h *TradeHandler) AdminCreateStock(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	var req model.AdminStockUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Symbol == "" || req.Name == "" || req.Price <= 0 {
		http.Error(w, "symbol, name and positive price are required", http.StatusBadRequest)
		return
	}
	if req.Series == "" {
		req.Series = "EQ"
	}

	if err := h.store.AdminCreateStock(req); err != nil {
		http.Error(w, "Failed to create stock", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Stock created/updated"})
}

func (h *TradeHandler) AdminUpdateStock(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	stockID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid stock id", http.StatusBadRequest)
		return
	}

	var req model.AdminStockUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	existing, err := h.store.GetStockByID(stockID)
	if err != nil {
		if errors.Is(err, store.ErrStockNotFound) {
			http.Error(w, "Stock not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch stock", http.StatusInternalServerError)
		return
	}

	if req.Symbol == "" {
		req.Symbol = existing.Symbol
	}
	if req.Name == "" {
		req.Name = existing.Name
	}
	if req.Price <= 0 {
		req.Price = existing.Price
	}
	if req.Series == "" {
		if existing.Series != "" {
			req.Series = existing.Series
		} else {
			req.Series = "EQ"
		}
	}

	err = h.store.AdminUpdateStock(stockID, req)
	if err != nil {
		if errors.Is(err, store.ErrStockNotFound) {
			http.Error(w, "Stock not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update stock", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Stock updated"})
}

func (h *TradeHandler) AdminDeleteStock(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	stockID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid stock id", http.StatusBadRequest)
		return
	}

	err = h.store.AdminDeleteStock(stockID)
	if err != nil {
		if errors.Is(err, store.ErrStockNotFound) {
			http.Error(w, "Stock not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete stock", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TradeHandler) GetTopStocks(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	stocks, err := h.store.GetTopStocks(100)
	if err != nil {
		http.Error(w, "Failed to fetch top stocks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"stocks": stocks, "count": len(stocks)})
}

func (h *TradeHandler) GetAllOrdersAdmin(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	orders, err := h.store.GetAllOrders(500)
	if err != nil {
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"orders": orders, "count": len(orders)})
}

// GetMarketStatus returns current market status
func (h *TradeHandler) GetMarketStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.store.GetMarketStatus()
	if err != nil {
		http.Error(w, "Failed to fetch market status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// StartMarket opens the market for trading (admin only)
func (h *TradeHandler) StartMarket(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	var (
		status *store.MarketStatus
		err    error
	)
	if h.marketService != nil {
		status, err = h.marketService.StartMarket()
	} else {
		status, err = h.store.StartMarket()
	}
	if err != nil {
		http.Error(w, "Failed to start market", http.StatusInternalServerError)
		return
	}

	// Broadcast market status to all WebSocket clients
	if h.broadcaster != nil {
		h.broadcaster.PublishMarketStatus(status.IsOpen)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Market started",
		"status":  status,
	})
}

// StopMarket closes the market for trading (admin only)
func (h *TradeHandler) StopMarket(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	var (
		status *store.MarketStatus
		err    error
	)
	if h.marketService != nil {
		status, err = h.marketService.StopMarket()
	} else {
		status, err = h.store.StopMarket()
	}
	if err != nil {
		http.Error(w, "Failed to stop market", http.StatusInternalServerError)
		return
	}

	// Broadcast market status to all WebSocket clients
	if h.broadcaster != nil {
		h.broadcaster.PublishMarketStatus(status.IsOpen)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Market stopped",
		"status":  status,
	})
}

func (h *TradeHandler) GetWatchlist(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	watchlist, err := h.store.GetWatchlistByUser(user.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch watchlist", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"watchlist": watchlist, "count": len(watchlist)})
}

func (h *TradeHandler) AddWatchlist(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		StockID       uuid.UUID `json:"stock_id"`
		WatchlistName string    `json:"watchlist_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.StockID == uuid.Nil {
		http.Error(w, "stock_id is required", http.StatusBadRequest)
		return
	}

	watchlistID, err := h.store.AddWatchlistItem(user.UserID, req.StockID, req.WatchlistName)
	if err != nil {
		log.Printf("Error adding to watchlist: %v", err)
		http.Error(w, "Failed to add watchlist item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"message": "Added to watchlist", "watchlist_id": watchlistID})
}

func (h *TradeHandler) RemoveWatchlist(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	watchlistID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid watchlist id", http.StatusBadRequest)
		return
	}

	err = h.store.RemoveWatchlistItem(user.UserID, watchlistID)
	if err != nil {
		if errors.Is(err, store.ErrWatchlistItemNotFound) {
			http.Error(w, "Watchlist item not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to remove watchlist item", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TradeHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		StockID     uuid.UUID `json:"stock_id"`
		TargetPrice float64   `json:"target_price"`
		Direction   string    `json:"direction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.StockID == uuid.Nil || req.TargetPrice <= 0 || (req.Direction != "above" && req.Direction != "below") {
		http.Error(w, "stock_id, positive target_price and direction(above|below) are required", http.StatusBadRequest)
		return
	}

	alert, err := h.store.CreateAlert(user.UserID, req.StockID, req.TargetPrice, req.Direction)
	if err != nil {
		http.Error(w, "Failed to create alert", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(alert)
}

func (h *TradeHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	alerts, err := h.store.GetAlertsByUser(user.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch alerts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"alerts": alerts, "count": len(alerts)})
}

func (h *TradeHandler) DeleteAlert(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	alertID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid alert id", http.StatusBadRequest)
		return
	}

	err = h.store.DeleteAlert(user.UserID, alertID)
	if err != nil {
		if errors.Is(err, store.ErrAlertNotFound) {
			http.Error(w, "Alert not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete alert", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TradeHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	unreadOnly := r.URL.Query().Get("unread") == "true"
	notifications, err := h.store.GetNotificationsByUser(user.UserID, unreadOnly)
	if err != nil {
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"notifications": notifications, "count": len(notifications)})
}

func (h *TradeHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user, err := h.getRequester(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	notificationID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid notification id", http.StatusBadRequest)
		return
	}

	if err := h.store.MarkNotificationRead(user.UserID, notificationID); err != nil {
		http.Error(w, "Failed to mark notification as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
