package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"backend-go/auth"
	"backend-go/db"
	"backend-go/handler"
	"backend-go/service"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// ── Load .env ────────────────────────────────────────────────────
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("warning: no .env file found, using system environment")
	}

	// ── DB ──────────────────────────────────────────────────────────
	if err := db.Init(); err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	log.Println("✓ postgres connected")

	// ── Redis ────────────────────────────────────────────────────────
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	handler.RedisClient = rdb
	log.Println("✓ redis client initialised at", redisAddr)

	// ── Initialize user wallets if needed ─────────────────────────────
	initializeExistingUserWallets()

	// ── Router ───────────────────────────────────────────────────────
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()

	// Health Check
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods(http.MethodGet)

	// ─────────────────────────────────────────────────────────────────
	// Public Routes (No Authentication Required)
	// ─────────────────────────────────────────────────────────────────

	// Auth routes
	api.HandleFunc("/auth/register", handler.Register).Methods(http.MethodPost)
	api.HandleFunc("/auth/login", handler.Login).Methods(http.MethodPost)
	api.HandleFunc("/auth/refresh", handler.Refresh).Methods(http.MethodPost)
	api.HandleFunc("/auth/logout", handler.Logout).Methods(http.MethodPost)
	api.HandleFunc("/auth/google", handler.GoogleLogin).Methods(http.MethodPost)

	// Public stock routes
	api.HandleFunc("/stocks", handler.GetAllStocks).Methods(http.MethodGet)
	api.HandleFunc("/stocks/search", handler.SearchStocks).Methods(http.MethodGet)
	api.HandleFunc("/stocks/{stock_id}", handler.GetStockByID).Methods(http.MethodGet)
	api.HandleFunc("/stocks/{stock_id}/stats", handler.GetStockStats).Methods(http.MethodGet)

	// ─────────────────────────────────────────────────────────────────
	// Protected Routes (Authentication Required)
	// ─────────────────────────────────────────────────────────────────

	protected := api.PathPrefix("").Subrouter()
	protected.Use(auth.JWTMiddleware)

	// Transaction Routes
	protected.HandleFunc("/transactions/buy", handler.BuyStock).Methods(http.MethodPost)
	protected.HandleFunc("/transactions/sell", handler.SellStock).Methods(http.MethodPost)
	protected.HandleFunc("/transactions/history", handler.GetTransactionHistory).Methods(http.MethodGet)

	// Portfolio Routes
	protected.HandleFunc("/portfolio", handler.GetPortfolio).Methods(http.MethodGet)

	// Wallet Routes
	protected.HandleFunc("/wallet", handler.GetWallet).Methods(http.MethodGet)

	// Watchlist Routes
	protected.HandleFunc("/watchlist", handler.GetWatchlist).Methods(http.MethodGet)
	protected.HandleFunc("/watchlist", handler.AddToWatchlist).Methods(http.MethodPost)
	protected.HandleFunc("/watchlist/{watchlist_id}", handler.RemoveFromWatchlist).Methods(http.MethodDelete)

	// Admin Routes (Stock Management)
	protected.HandleFunc("/admin/stocks", handler.CreateStock).Methods(http.MethodPost)
	protected.HandleFunc("/admin/stocks/{stock_id}", handler.UpdateStock).Methods(http.MethodPut)
	protected.HandleFunc("/admin/stocks/{stock_id}", handler.DeleteStock).Methods(http.MethodDelete)
	protected.HandleFunc("/admin/stocks/top", handler.GetTopStocks).Methods(http.MethodGet)

	// ── Listen ───────────────────────────────────────────────────────
	addr := getenv("SERVER_ADDR", ":8080")
	log.Println("✓ server listening on", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// initializeExistingUserWallets creates wallets for users who don't have one
func initializeExistingUserWallets() {
	ctx := context.Background()
	rows, err := db.Pool.Query(ctx, `
		SELECT u.user_id
		FROM users u
		LEFT JOIN wallet w ON u.user_id = w.user_id
		WHERE w.wallet_id IS NULL
	`)
	if err != nil {
		log.Println("warning: failed to check for users without wallets:", err)
		return
	}
	defer rows.Close()

	initialBalance := 100000.0 // Default initial balance for trading

	count := 0
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			log.Println("warning: failed to scan user_id:", err)
			continue
		}

		if err := service.InitializeWallet(ctx, userID, initialBalance); err != nil {
			log.Printf("warning: failed to initialize wallet for user %s: %v\n", userID, err)
			continue
		}
		count++
	}

	if count > 0 {
		log.Printf("✓ initialized %d wallets with %.2f balance\n", count, initialBalance)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
