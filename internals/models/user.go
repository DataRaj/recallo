package models

import (
	"database/sql"
	"errors"
	"time"

	"recallo/db"
)

type User struct {
	ID                          int64     `json:"id"`
	Name                        string    `json:"name"`
	Email                       string    `json:"email"`
	Password                    string    `json:"-"`
	AvatarURL                   string    `json:"avatar_url,omitempty"`
	RefreshTokenForWeb          string    `json:"-"`
	RefreshTokenForWebUpdatedAt time.Time `json:"-"`
	RefreshTokenForApp          string    `json:"-"`
	RefreshTokenForAppUpdatedAt time.Time `json:"-"`
	CreatedAt                   time.Time `json:"created_at"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

func GetUserByEmail(email string) (*User, error) {
	u := &User{}

	query := `
		SELECT
			id,
			name,
			email,
			password,
			COALESCE(refresh_token_web, ''),
			COALESCE(refresh_token_web_updated_at, '1970-01-01'),
			COALESCE(refresh_token_mobile, ''),
			COALESCE(refresh_token_mobile_updated_at, '1970-01-01'),
			created_at,
			updated_at
		FROM users
		WHERE email = $1
	`
	err := db.DB.QueryRow(query, email).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Password,
		&u.RefreshTokenForWeb,
		&u.RefreshTokenForWebUpdatedAt,
		&u.RefreshTokenForApp,
		&u.RefreshTokenForAppUpdatedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return u, nil
}

func CreateUser(name, email, password string) (*User, error) {
	u := &User{}

	query := `
		INSERT INTO users (
			name,
			email,
			password
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			created_at,
			updated_at
	`

	err := db.DB.QueryRow(
		query,
		name,
		email,
		password,
	).Scan(
		&u.ID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	u.Name = name
	u.Email = email
	u.Password = password

	return u, nil
}

func GetUserByID(userID int64) (*User, error) {
	u := &User{}

	query := `
		SELECT
			id,
			name,
			email,
			password,
			COALESCE(refresh_token_web, ''),
			COALESCE(refresh_token_web_updated_at, '1970-01-01'),
			COALESCE(refresh_token_mobile, ''),
			COALESCE(refresh_token_mobile_updated_at, '1970-01-01'),
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`
	err := db.DB.QueryRow(query, userID).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Password,
		&u.RefreshTokenForWeb,
		&u.RefreshTokenForWebUpdatedAt,
		&u.RefreshTokenForApp,
		&u.RefreshTokenForAppUpdatedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return u, nil
}

func (u User) ToMap() map[string]any {
	return map[string]any{
		"id":    u.ID,
		"name":  u.Name,
		"email": u.Email,
	}
}
