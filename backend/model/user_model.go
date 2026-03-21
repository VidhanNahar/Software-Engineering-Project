package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	UserID          uuid.UUID `json:"user_id"`
	UserName        string    `json:"name"`
	EmailID         string    `json:"email_id"`
	Password        string    `json:"password"`
	AadharID        *string   `json:"aadhar_id"`
	PanID           *string   `json:"pan_id"`
	PhoneNumber     *int64    `json:"phone_number"`
	DateOfBirth     time.Time `json:"date_of_birth"`
	IsVerifiedEmail bool      `json:"is_verified_email"`
}
