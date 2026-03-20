package controller

import (
	"backend-go/model"
	"backend-go/store"
	"database/sql"
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	store *store.Store
}

func NewUserHandler(s *store.Store) *UserHandler {
	return &UserHandler{store: s}
}

func GenerateOTP() string {
	num := rand.Intn(900000) + 100000
	return strconv.Itoa(num)
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user model.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	existingUser, err := h.store.GetUserByEmail(user.EmailID)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if existingUser != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error while hashing password: "+err.Error(), http.StatusInternalServerError)
		return
	}

	user.Password = string(hashedPassword)

	// TODO Generate an OTP to verify email
	otp := GenerateOTP()

	err = h.store.SetOTP(user.UserID, otp)
	if err != nil {
		http.Error(w, "Error setting OTP: "+err.Error(), http.StatusInternalServerError)
		return
	}

	user.Password = ""
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmailID string `json:"email_id"`
		OTP     string `json:"otp"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.store.GetUserByEmail(req.EmailID)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if user.IsVerifiedEmail {
		http.Error(w, "User already verified", http.StatusBadRequest)
		return
	}

	isValid, err := h.store.ValidateOTP(user.UserID, req.OTP)
	if err != nil {
		http.Error(w, "Error checking OTP: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !isValid {
		http.Error(w, "Invalid or expired OTP", http.StatusBadRequest)
		return
	}

	err = h.store.VerifyUser(user.UserID)
	if err != nil {
		http.Error(w, "Error verifying OTP: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.store.DeleteOTP(user.UserID)
	if err != nil {
		http.Error(w, "Error deleting OTP: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Email verified successfully. Please log in.",
	})
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var loginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&loginReq)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.store.GetUserByEmail(loginReq.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid email", http.StatusBadRequest)
		} else if err.Error() == "user not verified" {
			http.Error(w, "Account not verified. Please check your email.", http.StatusUnauthorized)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password))
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.UserID,
		"exp":     time.Now().Add(time.Hour * 12).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		http.Error(w, "Error signing JWT token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}
