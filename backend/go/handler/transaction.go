package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"backend-go/auth"
	"backend-go/service"
)

// BuyStockRequest represents a request to buy stocks
type BuyStockRequest struct {
	StockID  string  `json:"stock_id"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// SellStockRequest represents a request to sell stocks
type SellStockRequest struct {
	StockID  string  `json:"stock_id"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// TransactionResponse represents the response after a transaction
type TransactionResponse struct {
	OrderID          string  `json:"order_id"`
	Status           string  `json:"status"`
	Message          string  `json:"message"`
	RemainingBalance float64 `json:"remaining_balance,omitempty"`
	TotalCost        float64 `json:"total_cost,omitempty"`
	TotalProceeds    float64 `json:"total_proceeds,omitempty"`
	Quantity         int     `json:"quantity,omitempty"`
	PricePerStock    float64 `json:"price_per_stock,omitempty"`
}

// BuyStock handles buy stock requests
func BuyStock(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req BuyStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.StockID == "" || req.Quantity <= 0 || req.Price <= 0 {
		writeError(w, http.StatusBadRequest, "stock_id, quantity, and price are required and must be positive")
		return
	}

	ctx := context.Background()
	transaction, err := service.BuyStock(ctx, userID, req.StockID, req.Quantity, req.Price)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get updated wallet
	wallet, err := service.GetUserWallet(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch wallet")
		return
	}

	response := TransactionResponse{
		OrderID:          transaction.OrderID,
		Status:           string(transaction.Status),
		Message:          "Stock purchase successful",
		RemainingBalance: wallet.AvailableBalance,
		TotalCost:        transaction.TotalAmount,
		Quantity:         transaction.Quantity,
		PricePerStock:    transaction.PricePerStock,
	}

	writeJSON(w, http.StatusCreated, response)
}

// SellStock handles sell stock requests
func SellStock(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req SellStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.StockID == "" || req.Quantity <= 0 || req.Price <= 0 {
		writeError(w, http.StatusBadRequest, "stock_id, quantity, and price are required and must be positive")
		return
	}

	ctx := context.Background()
	transaction, err := service.SellStock(ctx, userID, req.StockID, req.Quantity, req.Price)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get updated wallet
	wallet, err := service.GetUserWallet(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch wallet")
		return
	}

	response := TransactionResponse{
		OrderID:          transaction.OrderID,
		Status:           string(transaction.Status),
		Message:          "Stock sale successful",
		RemainingBalance: wallet.Balance,
		TotalProceeds:    transaction.TotalAmount,
		Quantity:         transaction.Quantity,
		PricePerStock:    transaction.PricePerStock,
	}

	writeJSON(w, http.StatusCreated, response)
}

// GetPortfolio retrieves the user's portfolio
func GetPortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	ctx := context.Background()
	holdings, err := service.GetUserPortfolio(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch portfolio")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"holdings": holdings,
		"count":    len(holdings),
	})
}

// GetWallet retrieves the user's wallet information
func GetWallet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	ctx := context.Background()
	wallet, err := service.GetUserWallet(ctx, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "wallet not found")
		return
	}

	writeJSON(w, http.StatusOK, wallet)
}

// GetTransactionHistory retrieves the user's transaction history
func GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse query parameters
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	ctx := context.Background()
	transactions, err := service.GetTransactionHistory(ctx, userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch transaction history")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": transactions,
		"count":        len(transactions),
		"limit":        limit,
		"offset":       offset,
	})
}
