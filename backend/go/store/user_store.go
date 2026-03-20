package store

import (
	"backend-go/model"

	"github.com/google/uuid"
)

func (s *Store) CreateUser(user *model.User) error {
	err := s.db.QueryRow(
		`INSERT INTO user (name, email_id, password, aadhar_id, pan_id, phone_number, date_of_birth)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING user_id`,
		user.UserName, user.EmailID, user.Password, user.AadharID, user.PanID, user.PhoneNumber, user.DateOfBirth,
	).Scan(&user.UserID)

	return err
}

func (s *Store) GetUsers() ([]model.User, error) {
	var users []model.User

	rows, err := s.db.Query(
		`SELECT * FROM user`,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.UserID, &user.UserName, &user.EmailID, &user.Password, &user.AadharID, &user.PanID, &user.DateOfBirth, &user.IsVerifiedEmail); err != nil {
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
		`SELECT * FROM user WHERE user_id = $1`, userID,
	).Scan(&user.UserID, &user.UserName, &user.EmailID, &user.Password, &user.AadharID, &user.PanID, &user.DateOfBirth, &user.IsVerifiedEmail)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByEmail(emailID string) (*model.User, error) {
	var user model.User

	err := s.db.QueryRow(
		`SELECT * FROM user WHERE user_id = $1`, emailID,
	).Scan(&user.UserID, &user.UserName, &user.EmailID, &user.Password, &user.AadharID, &user.PanID, &user.DateOfBirth, &user.IsVerifiedEmail)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) UpdateUserByID(userID uuid.UUID, user *model.User) error {
	_, err := s.db.Exec(
		`UPDATE user
		SET name = $1, password = $2, phone_number = $3, is_verified_email = $4
		WHERE user_id = $5`,
		user.UserName, user.Password, user.PhoneNumber, user.IsVerifiedEmail, user.UserID,
	)
	return err
}

func (s *Store) DeleteUserByID(userID uuid.UUID) error {
	_, err := s.db.Exec(
		`DELETE FROM user where user_id = $1`, userID,
	)
	return err
}
