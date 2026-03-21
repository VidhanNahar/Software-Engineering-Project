package store

import (
	"backend-go/model"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (s *Store) CreateUser(user *model.User) error {
	err := s.db.QueryRow(
		`INSERT INTO users (name, email_id, password, aadhar_id, pan_id, phone_number, date_of_birth)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING user_id`,
		user.UserName, user.EmailID, user.Password, user.AadharID, user.PanID, user.PhoneNumber, user.DateOfBirth,
	).Scan(&user.UserID)

	return err
}

func (s *Store) GetUsers() ([]model.User, error) {
	var users []model.User

	rows, err := s.db.Query(
		`SELECT * FROM users`,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.UserID, &user.UserName, &user.EmailID, &user.Password, &user.AadharID, &user.PanID, &user.PhoneNumber, &user.DateOfBirth, &user.IsVerifiedEmail); err != nil {
			return nil, err
		}
		user.Password = ""
		users = append(users, user)
	}
	return users, nil
}

func (s *Store) GetUserByID(userID uuid.UUID) (*model.User, error) {
	var user model.User

	err := s.db.QueryRow(
		`SELECT * FROM users WHERE user_id = $1`, userID,
	).Scan(&user.UserID, &user.UserName, &user.EmailID, &user.Password, &user.AadharID, &user.PanID, &user.PhoneNumber, &user.DateOfBirth, &user.IsVerifiedEmail)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByEmail(emailID string) (*model.User, error) {
	var user model.User

	err := s.db.QueryRow(
		`SELECT * FROM users WHERE email_id = $1`, emailID,
	).Scan(&user.UserID, &user.UserName, &user.EmailID, &user.Password, &user.AadharID, &user.PanID, &user.PhoneNumber, &user.DateOfBirth, &user.IsVerifiedEmail)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) UpdateUserByID(userID uuid.UUID, user *model.User) error {
	_, err := s.db.Exec(
		`UPDATE users
		SET name = $1, password = $2, phone_number = $3, is_verified_email = $4
		WHERE user_id = $5`,
		user.UserName, user.Password, user.PhoneNumber, user.IsVerifiedEmail, user.UserID,
	)
	return err
}

func (s *Store) DeleteUserByID(userID uuid.UUID) error {
	_, err := s.db.Exec(
		`DELETE FROM users WHERE user_id = $1`, userID,
	)
	return err
}

func (s *Store) SetOTP(userID uuid.UUID, otp string) error {
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", userID.String())

	err := s.rdb.Set(ctx, key, otp, 10*time.Minute).Err()
	return err
}

func (s *Store) ValidateOTP(userID uuid.UUID, otp string) (bool, error) {
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", userID.String())

	storedOTP, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return storedOTP == otp, nil
}

func (s *Store) VerifyUser(userID uuid.UUID) error {
	_, err := s.db.Exec(
		`UPDATE users
		SET is_verified_email = true WHERE user_id = $1`, userID,
	)
	return err
}

func (s *Store) DeleteOTP(userID uuid.UUID) error {
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", userID.String())
	return s.rdb.Del(ctx, key).Err()
}
