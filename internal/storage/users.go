package storage

import (
	"database/sql"
	"errors"
	"fmt"
)

type UserWithPasswordHash struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	IsActive     bool
}

func GetUserByUsername(db *sql.DB, username string) (UserWithPasswordHash, error) {
	var user UserWithPasswordHash
	var isActive int

	err := db.QueryRow(
		`SELECT id, username, password_hash, role, is_active
		 FROM users
		 WHERE username = ?`,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&isActive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserWithPasswordHash{}, ErrUserNotFound
		}

		return UserWithPasswordHash{}, fmt.Errorf("get user by username: %w", err)
	}

	user.IsActive = isActive == 1

	return user, nil
}

func GetUserByID(db *sql.DB, userID int64) (UserWithPasswordHash, error) {
	var user UserWithPasswordHash
	var isActive int

	err := db.QueryRow(
		`SELECT id, username, password_hash, role, is_active
		 FROM users
		 WHERE id = ?`,
		userID,
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&isActive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserWithPasswordHash{}, ErrUserNotFound
		}

		return UserWithPasswordHash{}, fmt.Errorf("get user by id: %w", err)
	}

	user.IsActive = isActive == 1

	return user, nil
}

var ErrUserNotFound = errors.New("user not found")

func GetUsernameByID(db *sql.DB, userID int64) (string, error) {
	var username string

	err := db.QueryRow(
		`SELECT username FROM users WHERE id = ?`,
		userID,
	).Scan(&username)
	if err != nil {
		return "", fmt.Errorf("get username by id: %w", err)
	}

	return username, nil
}
