package main

import (
	"backend-go/controller"
	"backend-go/database"
	"backend-go/middleware"
	"backend-go/store"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load env from the current directory
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	// Connect to database
	db, err := database.Connect(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		fmt.Println("Failed to connect to database: ", err)
		return
	}
	fmt.Println("Successfully connected to database")
	defer database.Close(db)

	if err := database.RunAutoSeeder(db); err != nil {
		log.Println("Warning: auto seeder failed:", err)
	} else {
		log.Println("Auto seeder completed")
	}

	// Create a redis client
	rdb, err := database.ConnectRedis()
	if err != nil {
		fmt.Println("Failed to connect to redis: ", err)
		return
	}
	fmt.Println("Successfully connected to redis")
	defer database.CloseRedis(rdb)

	// Create a store
	s := store.NewStore(db, rdb)
	adminEmail := os.Getenv("DEFAULT_ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@papertrade.local"
	}
	adminPassword := os.Getenv("DEFAULT_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "Admin@123"
	}
	adminName := os.Getenv("DEFAULT_ADMIN_NAME")
	if adminName == "" {
		adminName = "System Admin"
	}

	if err := s.EnsureDefaultAdmin(adminEmail, adminPassword, adminName); err != nil {
		log.Println("Warning: failed to ensure default admin:", err)
	}

	// Create a http router
	r := mux.NewRouter()

	u := controller.NewUserHandler(s)
	t := controller.NewTradeHandler(s)
	rt := controller.NewRealtimeHandler(s)

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")

	r.HandleFunc("/auth/register", u.CreateUser).Methods("POST")
	r.HandleFunc("/auth/login", u.Login).Methods("POST")
	r.HandleFunc("/auth/verify", u.VerifyEmail).Methods("POST")
	r.HandleFunc("/auth/refresh", u.RefreshToken).Methods("POST")
	r.HandleFunc("/auth/logout", u.Logout).Methods("POST")
	r.HandleFunc("/api/auth/register", u.CreateUser).Methods("POST")
	r.HandleFunc("/api/auth/login", u.Login).Methods("POST")
	r.HandleFunc("/api/auth/verify", u.VerifyEmail).Methods("POST")
	r.HandleFunc("/api/auth/refresh", u.RefreshToken).Methods("POST")
	r.HandleFunc("/api/auth/logout", u.Logout).Methods("POST")
	r.HandleFunc("/stocks", t.GetStocks).Methods("GET")
	r.HandleFunc("/api/stocks", t.GetStocks).Methods("GET")
	r.HandleFunc("/api/stocks/search", t.SearchStocks).Methods("GET")
	r.HandleFunc("/api/stocks/{id}", t.GetStockByID).Methods("GET")
	r.HandleFunc("/api/stocks/symbol/{symbol}", t.GetStockBySymbol).Methods("GET")
	r.HandleFunc("/api/stocks/{id}/history", t.GetStockHistory).Methods("GET")
	r.HandleFunc("/api/stocks/{id}/stats", t.GetStockStats).Methods("GET")
	r.HandleFunc("/ws/stocks", rt.StocksStream).Methods("GET")

	api := r.PathPrefix("/api").Subrouter()
	api.Use(middleware.AuthMiddleware)

	api.HandleFunc("/user", u.GetUsers).Methods("GET")
	api.HandleFunc("/user/{id}", u.GetUserByID).Methods("GET")
	api.HandleFunc("/user/{id}", u.UpdateUserByID).Methods("PUT")
	api.HandleFunc("/user/{id}", u.DeleteUserByID).Methods("DELETE")
	api.HandleFunc("/user/kyc/complete", u.CompleteKYC).Methods("POST")
	api.HandleFunc("/trade/buy", t.BuyStock).Methods("POST")
	api.HandleFunc("/trade/sell", t.SellStock).Methods("POST")
	api.HandleFunc("/transactions/buy", t.BuyStock).Methods("POST")
	api.HandleFunc("/transactions/sell", t.SellStock).Methods("POST")
	api.HandleFunc("/portfolio", t.GetPortfolio).Methods("GET")
	api.HandleFunc("/orders", t.GetOrders).Methods("GET")
	api.HandleFunc("/transactions/history", t.GetOrders).Methods("GET")
	api.HandleFunc("/wallet", t.GetWallet).Methods("GET")
	api.HandleFunc("/watchlist", t.GetWatchlist).Methods("GET")
	api.HandleFunc("/watchlist", t.AddWatchlist).Methods("POST")
	api.HandleFunc("/watchlist/{id}", t.RemoveWatchlist).Methods("DELETE")
	api.HandleFunc("/alerts", t.GetAlerts).Methods("GET")
	api.HandleFunc("/alerts", t.CreateAlert).Methods("POST")
	api.HandleFunc("/alerts/{id}", t.DeleteAlert).Methods("DELETE")
	api.HandleFunc("/notifications", t.GetNotifications).Methods("GET")
	api.HandleFunc("/notifications/{id}/read", t.MarkNotificationRead).Methods("PATCH")

	api.HandleFunc("/admin/stocks", t.AdminCreateStock).Methods("POST")
	api.HandleFunc("/admin/stocks/{id}", t.AdminUpdateStock).Methods("PUT")
	api.HandleFunc("/admin/stocks/{id}", t.AdminDeleteStock).Methods("DELETE")
	api.HandleFunc("/admin/stocks/top", t.GetTopStocks).Methods("GET")
	api.HandleFunc("/admin/orders", t.GetAllOrdersAdmin).Methods("GET")

	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
