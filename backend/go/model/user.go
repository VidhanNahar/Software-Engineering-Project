package model

import "time"

type User struct {
	UserID          string    `json:"user_id"`
	Name            string    `json:"name"`
	EmailID         string    `json:"email_id"`
	Password        string    `json:"password"`
	AadharID        string    `json:"aadhar_id"`
	PanID           string    `json:"pan_id"`
	PhoneNumber     string    `json:"phone_number"`
	DateOfBirth     time.Time `json:"date_of_birth"`
	IsVerifiedEmail bool      `json:"is_verified_email"`
}