package controller

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// HealthHandler provides health check for load balancers
type HealthHandler struct {
	db  *sql.DB
	rdb *redis.Client
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(db *sql.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:  db,
		rdb: rdb,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Database  string            `json:"database"`
	Cache     string            `json:"cache"`
	Timestamp string            `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Details   map[string]string `json:"details"`
}

var startTime = time.Now()

// Check performs a comprehensive health check
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    time.Since(startTime).String(),
		Details:   make(map[string]string),
	}

	// Check database health
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		response.Database = "unhealthy"
		response.Details["db_error"] = err.Error()
		response.Status = "degraded"
	} else {
		response.Database = "healthy"
	}

	// Check Redis health
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	if err := h.rdb.Ping(ctx2).Err(); err != nil {
		response.Cache = "unhealthy"
		response.Details["redis_error"] = err.Error()
		response.Status = "degraded"
	} else {
		response.Cache = "healthy"
	}

	// If both DB and Redis are down, mark as unhealthy
	if response.Database == "unhealthy" && response.Cache == "unhealthy" {
		response.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else if response.Status == "degraded" {
		// Degraded but still accepting requests
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Readiness checks if service is ready to handle requests
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check if database is accessible
	if err := h.db.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not_ready", "reason": "database unavailable"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
