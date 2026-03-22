package controller

import (
	"backend-go/middleware"
	"backend-go/model"
	"backend-go/store"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

// TradeHandler serves trade-related APIs.
type TradeHandler struct {
	store *store.Store
}

func NewTradeHandler(s *store.Store) *TradeHandler {
	return &TradeHandler{store: s}
}

// BuyStock executes a buy transaction for authenticated user.
func (h *TradeHandler) BuyStock(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.TradeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.store.ExecuteBuyTx(r.Context(), store.TradeRequest{UserID: userID, StockID: req.StockID, Quantity: req.Quantity})
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
		"message":  "Buy order executed",
		"stock_id": req.StockID,
		"quantity": req.Quantity,
	})
}

// SellStock executes a sell transaction for authenticated user.
func (h *TradeHandler) SellStock(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.TradeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.store.ExecuteSellTx(r.Context(), store.TradeRequest{UserID: userID, StockID: req.StockID, Quantity: req.Quantity})
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
		"message":  "Sell order executed",
		"stock_id": req.StockID,
		"quantity": req.Quantity,
	})
}

// GetStocks returns all stock quotes.
func (h *TradeHandler) GetStocks(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.store.GetStocks()
	if err != nil {
		http.Error(w, "Failed to fetch stocks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stocks)
}

// GetPortfolio returns authenticated user's holdings.
func (h *TradeHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	positions, err := h.store.GetPortfolioByUser(userID)
	if err != nil {
		http.Error(w, "Failed to fetch portfolio", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(positions)
}

// GetOrders returns authenticated user's transaction history.
func (h *TradeHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.store.GetOrdersByUser(userID)
	if err != nil {
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
