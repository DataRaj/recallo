package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"gotel/db"
	"gotel/internals/middleware"
)

func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.Strict().EncodeToString(b), nil
}

func UpdateRefreshToken(userID int64, platform, refreshToken string) error {
	db, err := db.GetDB()
	if err != nil {
		return err
	}

	now := time.Now()

	switch platform {
	case middleware.PlatformWeb:
		_, err = db.Exec("UPDATE users SET refresh_token_web = ?, refresh_token_web_at = ? WHERE id = ?", refreshToken, now, userID)

	case middleware.PlatformMobile:
		_, err = db.Exec("UPDATE users SET refresh_token_mobile = ?, refresh_token_mobile_at = ? WHERE id = ?", refreshToken, now, userID)

	default:
		return errors.New("invalid platform")
	}

	return err
}

func DeleteUserRefreshToken(userId int64, platform string) error {
	db, err := db.GetDB()
	if err != nil {
		return err
	}

	switch platform {
	case middleware.PlatformWeb:
		_, err = db.Exec("UPDATE users SET refresh_token_web = NULL, refresh_token_web_at = NULL WHERE id = ?", userId)
	case middleware.PlatformMobile:
		_, err = db.Exec("UPDATE users SET refresh_token_mobile = NULL, refresh_token_mobile_at = NULL WHERE id = ?", userId)
	default:
		return errors.New("invalid platform")
	}

	return err
}
