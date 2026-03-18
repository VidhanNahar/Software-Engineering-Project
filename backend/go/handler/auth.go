package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"backend-go/auth"
	"backend-go/db"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ── Redis client (set by main) ────────────────────────────────────────────────

var RedisClient *redis.Client

// ── Register ─────────────────────────────────────────────────────────────────

type registerRequest struct {
	Name            string `json:"name"`
	EmailID         string `json:"email_id"`
	Password        string `json:"password"`
	AadharID        string `json:"aadhar_id"`
	PanID           string `json:"pan_id"`
	PhoneNumber     string `json:"phone_number"`
	DateOfBirth     string `json:"date_of_birth"` // "YYYY-MM-DD"
	IsVerifiedEmail bool   `json:"is_verified_email"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.EmailID == "" || req.Password == "" || req.DateOfBirth == "" {
		writeError(w, http.StatusBadRequest, "name, email_id, password and date_of_birth are required")
		return
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		writeError(w, http.StatusBadRequest, "date_of_birth must be YYYY-MM-DD")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	var userID string
	err = db.Pool.QueryRow(context.Background(),
		`INSERT INTO users (name, email_id, password, aadhar_id, pan_id, phone_number, date_of_birth, is_verified_email)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING user_id`,
		req.Name, req.EmailID, string(hash),
		nullStr(req.AadharID), nullStr(req.PanID), nullStr(req.PhoneNumber),
		dob, req.IsVerifiedEmail,
	).Scan(&userID)
	if err != nil {
		writeError(w, http.StatusConflict, "user already exists or invalid data")
		return
	}

	accessToken, err := auth.GenerateAccessToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"user_id":       userID,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// ── Login ─────────────────────────────────────────────────────────────────────

type loginRequest struct {
	EmailID  string `json:"email_id"`
	Password string `json:"password"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.EmailID == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email_id and password are required")
		return
	}

	var userID, hashedPassword string
	err := db.Pool.QueryRow(context.Background(),
		`SELECT user_id, password FROM users WHERE email_id = $1`,
		req.EmailID,
	).Scan(&userID, &hashedPassword)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	accessToken, err := auth.GenerateAccessToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id":       userID,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// ── Refresh ───────────────────────────────────────────────────────────────────

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check if token is blacklisted in Redis
	if RedisClient != nil {
		if val, _ := RedisClient.Get(context.Background(), "bl:"+req.RefreshToken).Result(); val != "" {
			writeError(w, http.StatusUnauthorized, "token has been revoked")
			return
		}
	}

	claims, err := auth.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	accessToken, err := auth.GenerateAccessToken(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"access_token": accessToken,
	})
}

// ── Logout ────────────────────────────────────────────────────────────────────

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	claims, err := auth.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid refresh token")
		return
	}

	// Blacklist the token in Redis until it naturally expires
	if RedisClient != nil {
		ttl := time.Until(claims.ExpiresAt.Time)
		RedisClient.Set(context.Background(), "bl:"+req.RefreshToken, "1", ttl)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// ── util ──────────────────────────────────────────────────────────────────────

// nullStr returns nil for empty strings so Postgres treats optional fields as NULL.
func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ── Google OAuth ───────────────────────────────────────────────────────────────

type googleLoginRequest struct {
	Credential string `json:"credential"`
}

func GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Credential == "" {
		writeError(w, http.StatusBadRequest, "credential is required")
		return
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		writeError(w, http.StatusInternalServerError, "Google OAuth not configured")
		return
	}

	// Validate the Google ID token
	payload, err := idtoken.Validate(r.Context(), req.Credential, clientID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid Google token")
		return
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	if email == "" {
		writeError(w, http.StatusBadRequest, "email not provided by Google")
		return
	}
	if name == "" {
		name = email
	}

	// Upsert user: insert if not exists, retrieve user_id either way
	var userID string
	err = db.Pool.QueryRow(context.Background(),
		`INSERT INTO users (name, email_id, password, is_verified_email)
		 VALUES ($1, $2, '', true)
		 ON CONFLICT (email_id) DO UPDATE SET name = EXCLUDED.name
		 RETURNING user_id`,
		name, email,
	).Scan(&userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to upsert user")
		return
	}

	accessToken, err := auth.GenerateAccessToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id":       userID,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

