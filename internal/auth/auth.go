package auth

import (
	"database/sql"
	"errors"

	"northgate-srms/internal/storage"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

func AuthenticateUser(db *sql.DB, username string, password string) (User, error) {
	storedUser, err := storage.GetUserByUsername(db, username)
	if err != nil {
		return User{}, ErrInvalidCredentials
	}

	if !storedUser.IsActive {
		return User{}, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(storedUser.PasswordHash),
		[]byte(password),
	)
	if err != nil {
		return User{}, ErrInvalidCredentials
	}

	return User{
		ID:       storedUser.ID,
		Username: storedUser.Username,
		Role:     storedUser.Role,
	}, nil
}
