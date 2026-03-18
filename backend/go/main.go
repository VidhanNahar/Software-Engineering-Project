package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"backend-go/auth"
	"backend-go/db"
	"backend-go/handler"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

func main() {
	// ── DB ──────────────────────────────────────────────────────────
	if err := db.Init(); err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	log.Println("postgres connected")

	// ── Redis ────────────────────────────────────────────────────────
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	handler.RedisClient = rdb
	log.Println("redis client initialised at", redisAddr)

	// ── Router ───────────────────────────────────────────────────────
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()

	// Health
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods(http.MethodGet)

	// Auth routes (public)
	api.HandleFunc("/auth/register", handler.Register).Methods(http.MethodPost)
	api.HandleFunc("/auth/login", handler.Login).Methods(http.MethodPost)
	api.HandleFunc("/auth/refresh", handler.Refresh).Methods(http.MethodPost)
	api.HandleFunc("/auth/logout", handler.Logout).Methods(http.MethodPost)

	// Protected subrouter — add JWT middleware
	protected := api.PathPrefix("").Subrouter()
	protected.Use(auth.JWTMiddleware)
	// e.g. protected.HandleFunc("/users/me", handler.Me).Methods(http.MethodGet)

	// ── Listen ───────────────────────────────────────────────────────
	addr := getenv("SERVER_ADDR", ":8080")
	log.Println("server listening on", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
