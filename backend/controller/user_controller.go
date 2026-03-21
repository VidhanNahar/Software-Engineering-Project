package controller

import (
	"backend-go/middleware"
	"backend-go/model"
	"backend-go/store"
	"backend-go/utils"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	store *store.Store
}

func NewUserHandler(s *store.Store) *UserHandler {
	return &UserHandler{store: s}
}

func GenerateOTP() string {
	max := big.NewInt(900000)
	num, _ := rand.Int(rand.Reader, max)
	return strconv.FormatInt(num.Int64()+100000, 10)
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

	// Save user to database before generating OTP
	if err := h.store.CreateUser(&user); err != nil {
		http.Error(w, "Error saving user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	otp := GenerateOTP()

	err = h.store.SetOTP(user.UserID, otp)
	if err != nil {
		http.Error(w, "Error setting OTP: "+err.Error(), http.StatusInternalServerError)
		return
	}

	go func() {
		err := utils.SendOTP(user.EmailID, user.UserName, otp)
		if err != nil {
			fmt.Printf("Failed to send verification email to %s: %v\n", user.EmailID, err)
		}
	}()

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
		Email    string `json:"email_id"`
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
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if !user.IsVerifiedEmail {
		http.Error(w, "Account not verified. Please check your email.", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password))
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": user.UserID,
		"exp":    time.Now().Add(time.Hour * 12).Unix(),
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

func (h *UserHandler) UpdateUserByID(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	params := mux.Vars(r)

	userID, err := uuid.Parse(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if userID != requesterID {
		http.Error(w, "Unauthorized: you can edit only your profile", http.StatusForbidden)
		return
	}

	var reqUser model.User
	if err := json.NewDecoder(r.Body).Decode(&reqUser); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	existingUser, err := h.store.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if reqUser.UserName != "" {
		existingUser.UserName = reqUser.UserName
	}
	if reqUser.PhoneNumber != nil {
		existingUser.PhoneNumber = reqUser.PhoneNumber
	}
	if reqUser.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(reqUser.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}
		existingUser.Password = string(hashed)
	}

	if err := h.store.UpdateUserByID(userID, existingUser); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/type")
	json.NewEncoder(w).Encode("Updated user successfully")
}

func (h *UserHandler) DeleteUserByID(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	params := mux.Vars(r)

	userID, err := uuid.Parse(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if userID != requesterID {
		http.Error(w, "Unauthorized: you can delete only your profile", http.StatusForbidden)
		return
	}

	if err := h.store.DeleteUserByID(userID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.GetUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	userID, err := uuid.Parse(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := h.store.GetUserByID(userID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Clear out password hash before sending response
	user.Password = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
