package main

import (
	"backend-go/controller"
	"backend-go/database"
	"backend-go/market"
	"backend-go/middleware"
	"backend-go/store"
	"backend-go/utils"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	// Initialize async email queue (10 workers, 1000 job buffer)
	emailQueue := utils.NewEmailQueue(10, 1000)
	defer emailQueue.Stop()

	// Create a http router
	r := mux.NewRouter()
	allowedOrigins := map[string]struct{}{
		"http://20.193.252.172": {},
	}
	if frontendOrigin := os.Getenv("FRONTEND_ORIGIN"); frontendOrigin != "" {
		allowedOrigins[frontendOrigin] = struct{}{}
	}
	isAllowedOrigin := func(origin string) bool {
		if origin == "" {
			return false
		}
		if _, ok := allowedOrigins[origin]; ok {
			return true
		}

		parsedOrigin, err := url.Parse(origin)
		if err != nil {
			return false
		}

		hostname := strings.ToLower(parsedOrigin.Hostname())
		if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
			return true
		}

		return false
	}
	setCORSHeaders := func(w http.ResponseWriter, origin string) {
		if isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setCORSHeaders(w, r.Header.Get("Origin"))

			// 2. Handle the Preflight (OPTIONS) request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Setup rate limiter
	rateLimiter := middleware.NewRateLimitStore()

	// Apply middleware in order (from bottom to top in request pipeline)
	// IMPORTANT: SelectiveTimeoutMiddleware skips WS routes to allow long-lived connections
	r.Use(corsMiddleware)
	r.Use(middleware.SelectiveTimeoutMiddleware(30 * time.Second))
	r.Use(rateLimiter.RateLimitMiddleware(100)) // 100 requests per minute
	r.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w, r.Header.Get("Origin"))
		w.WriteHeader(http.StatusNoContent)
	})

	broadcaster := market.NewWebSocketBroadcaster()
	marketService := market.NewMarketService(s, broadcaster)
	log.Println("Market engine initialized")

	// Use mock price broadcaster instead of real market engine (no database hits)
	mockBroadcaster := market.NewMockPriceBroadcaster(broadcaster, s)

	// Load initial stocks and start mock broadcaster
	go func() {
		time.Sleep(1 * time.Second) // Wait for DB to be ready
		stocks, err := s.GetStocks()
		if err != nil {
			log.Println("Warning: could not load stocks for mock broadcaster:", err)
			return
		}
		mockBroadcaster.SetStocks(stocks)
		mockBroadcaster.Start()
		log.Println("✅ Mock price broadcaster started - prices will update every 2 seconds")
	}()

	u := controller.NewUserHandler(s, emailQueue)
	t := controller.NewTradeHandler(s, marketService, broadcaster)
	rt := controller.NewRealtimeHandler(s, broadcaster)
	h := controller.NewHealthHandler(db, rdb)

	// Routes for health checks (no auth required)
	r.HandleFunc("/health", h.Check).Methods("GET")
	r.HandleFunc("/readiness", h.Readiness).Methods("GET")

	r.HandleFunc("/auth/register", u.CreateUser).Methods("POST")
	r.HandleFunc("/auth/login", u.Login).Methods("POST")
	r.HandleFunc("/auth/verify", u.VerifyEmail).Methods("POST")
	r.HandleFunc("/auth/refresh", u.RefreshToken).Methods("POST")
	r.HandleFunc("/auth/logout", u.Logout).Methods("POST")
	r.HandleFunc("/auth/forgot-password", u.ForgotPassword).Methods("POST")
	r.HandleFunc("/auth/reset-password", u.ResetPassword).Methods("POST")
	r.HandleFunc("/stocks", t.GetStocks).Methods("GET")
	r.HandleFunc("/api/stocks", t.GetStocks).Methods("GET")
	r.HandleFunc("/api/stocks/search", t.SearchStocks).Methods("GET")
	r.HandleFunc("/api/stocks/{symbol}/ticks", t.GetStockTicksBySymbol).Methods("GET")
	r.HandleFunc("/api/stocks/{symbol}/candles", t.GetStockCandlesBySymbol).Methods("GET")
	r.HandleFunc("/api/stocks/{id}", t.GetStockByID).Methods("GET")
	r.HandleFunc("/api/stocks/symbol/{symbol}", t.GetStockBySymbol).Methods("GET")
	r.HandleFunc("/api/stocks/symbol/{symbol}/ticks", t.GetStockTicksBySymbol).Methods("GET")
	r.HandleFunc("/api/stocks/symbol/{symbol}/candles", t.GetStockCandlesBySymbol).Methods("GET")
	r.HandleFunc("/api/stocks/{id}/history", t.GetStockHistory).Methods("GET")
	r.HandleFunc("/api/stocks/{id}/stats", t.GetStockStats).Methods("GET")
	r.HandleFunc("/api/market/status", t.GetMarketStatus).Methods("GET")

	// WebSocket route - NO Methods() restriction (WebSocket upgrade is special HTTP)
	r.HandleFunc("/ws/stocks", rt.StocksStream)

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
	api.HandleFunc("/pending-orders", t.GetPendingOrders).Methods("GET")
	api.HandleFunc("/pending-orders/{order_id}", t.CancelPendingOrder).Methods("DELETE")
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
	api.HandleFunc("/admin/market/start", t.StartMarket).Methods("POST")
	api.HandleFunc("/admin/market/stop", t.StopMarket).Methods("POST")
	api.HandleFunc("/admin/pending-orders/release", t.ReleasePendingOrdersOnMarketClose).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server with timeouts for production
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	log.Printf("✅ Server starting on port %s...", port)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown: listen for interrupt signals (SIGINT, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal received
	<-sigChan

	log.Println("\n🛑 Shutdown signal received, gracefully shutting down...")

	// Create a context with 30-second timeout for graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("\n✅ Server shutdown complete")
}
