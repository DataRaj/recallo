package models

import (
	"database/sql"
	"errors"
	"time"

	"gotel/db"
)

type User struct {
	ID                          int64     `json:"id"`
	Name                        string    `json:"name"`
	Email                       string    `json:"email"`
	Password                    string    `json:"-"`
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
			refresh_token_for_web,
			refresh_token_for_web_updated_at,
			refresh_token_for_app,
			refresh_token_for_app_updated_at,
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
