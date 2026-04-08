package store

import (
	"backend-go/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const StarterWalletBalance = 500000.0

var ErrUserNotFound = errors.New("user not found")

const userSelectColumns = `
	user_id,
	name,
	email_id,
	password,
	role,
	aadhar_id,
	pan_id,
	phone_number,
	date_of_birth,
	is_verified_email,
	is_kyc_verified`

func (s *Store) CreateUser(user *model.User) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = tx.QueryRow(
		`INSERT INTO users (name, email_id, password, role, aadhar_id, pan_id, phone_number, date_of_birth, is_verified_email, is_kyc_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING user_id`,
		user.UserName, user.EmailID, user.Password, "guest", user.AadharID, user.PanID, user.PhoneNumber, user.DateOfBirth, false, false,
	).Scan(&user.UserID)
	if err != nil {
		return err
	}

	// Create wallet with 5L opening balance for paper-trading.
	_, err = tx.Exec(
		`INSERT INTO wallet (user_id, balance, locked_balance) VALUES ($1, $2, $3)`,
		user.UserID, StarterWalletBalance, 0.0,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) GetUsers() ([]model.User, error) {
	var users []model.User

	rows, err := s.db.Query(
		`SELECT ` + userSelectColumns + ` FROM users ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.UserID,
			&user.UserName,
			&user.EmailID,
			&user.Password,
			&user.Role,
			&user.AadharID,
			&user.PanID,
			&user.PhoneNumber,
			&user.DateOfBirth,
			&user.IsVerifiedEmail,
			&user.IsKYCVerified,
		); err != nil {
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
		`SELECT `+userSelectColumns+` FROM users WHERE user_id = $1`, userID,
	).Scan(
		&user.UserID,
		&user.UserName,
		&user.EmailID,
		&user.Password,
		&user.Role,
		&user.AadharID,
		&user.PanID,
		&user.PhoneNumber,
		&user.DateOfBirth,
		&user.IsVerifiedEmail,
		&user.IsKYCVerified,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByEmail(emailID string) (*model.User, error) {
	var user model.User

	err := s.db.QueryRow(
		`SELECT `+userSelectColumns+` FROM users WHERE email_id = $1`, emailID,
	).Scan(
		&user.UserID,
		&user.UserName,
		&user.EmailID,
		&user.Password,
		&user.Role,
		&user.AadharID,
		&user.PanID,
		&user.PhoneNumber,
		&user.DateOfBirth,
		&user.IsVerifiedEmail,
		&user.IsKYCVerified,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &user, nil
}

func (s *Store) UpdateUserByID(userID uuid.UUID, user *model.User) error {
	_, err := s.db.Exec(
		`UPDATE users
		SET name = $1, password = $2, phone_number = $3, is_verified_email = $4, role = $5, is_kyc_verified = $6,
			aadhar_id = $7, pan_id = $8
		WHERE user_id = $9`,
		user.UserName,
		user.Password,
		user.PhoneNumber,
		user.IsVerifiedEmail,
		user.Role,
		user.IsKYCVerified,
		user.AadharID,
		user.PanID,
		user.UserID,
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

func (s *Store) CompleteKYC(userID uuid.UUID, aadharID *string, panID *string) error {
	res, err := s.db.Exec(
		`UPDATE users
		SET aadhar_id = COALESCE($1, aadhar_id),
			pan_id = COALESCE($2, pan_id),
			is_kyc_verified = true,
			role = 'user'
		WHERE user_id = $3`,
		aadharID,
		panID,
		userID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *Store) GetWalletByUser(userID uuid.UUID) (float64, float64, error) {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	var balance float64
	var locked float64
	err = tx.QueryRow(`SELECT balance, locked_balance FROM wallet WHERE user_id = $1 FOR UPDATE`, userID).Scan(&balance, &locked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, ErrUserNotFound
		}
		return 0, 0, err
	}

	// Reconcile locked balance with active BUY limit orders (covers legacy orders created before lock logic).
	var expectedLocked float64
	err = tx.QueryRow(`
		SELECT COALESCE(SUM((po.quantity - po.filled_quantity) * po.limit_price *
			CASE WHEN UPPER(st.currency_code) = 'USD' THEN 83.5 ELSE 1 END), 0)
		FROM pending_orders po
		JOIN stock st ON st.stock_id = po.stock_id
		WHERE po.user_id = $1 AND po.order_type = 'BUY' AND po.status IN ('PENDING', 'PARTIALLY_FILLED')`, userID).Scan(&expectedLocked)
	if err != nil {
		return 0, 0, err
	}

	if expectedLocked != locked {
		delta := expectedLocked - locked
		if delta > 0 {
			if delta > balance {
				delta = balance
			}
			_, err = tx.Exec(`UPDATE wallet SET balance = balance - $1, locked_balance = locked_balance + $1 WHERE user_id = $2`, delta, userID)
			if err != nil {
				return 0, 0, err
			}
		} else {
			releaseAmount := -delta
			_, err = tx.Exec(`UPDATE wallet SET balance = balance + $1, locked_balance = GREATEST(locked_balance - $1, 0) WHERE user_id = $2`, releaseAmount, userID)
			if err != nil {
				return 0, 0, err
			}
		}

		err = tx.QueryRow(`SELECT balance, locked_balance FROM wallet WHERE user_id = $1`, userID).Scan(&balance, &locked)
		if err != nil {
			return 0, 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	return balance, locked, nil
}

func (s *Store) EnsureDefaultAdmin(email, password, name string) error {
	existing, err := s.GetUserByEmail(email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if existing != nil {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID uuid.UUID
	err = tx.QueryRow(
		`INSERT INTO users (name, email_id, password, role, date_of_birth, is_verified_email, is_kyc_verified)
		VALUES ($1, $2, $3, 'admin', $4, true, true)
		RETURNING user_id`,
		name,
		email,
		string(hashedPassword),
		time.Now().AddDate(-20, 0, 0),
	).Scan(&userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO wallet (user_id, balance, locked_balance) VALUES ($1, $2, $3)`, userID, StarterWalletBalance, 0.0)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) DeleteOTP(userID uuid.UUID) error {
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", userID.String())
	return s.rdb.Del(ctx, key).Err()
}
