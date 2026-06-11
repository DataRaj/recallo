package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey []byte

func InitJWT(key string) {
	jwtKey = []byte(key)
}

type CustomClaims struct {
	UserID    int64  `json:"user_id"`
	Name      string `json:"name"`
	XPlatform string `json:"X-Platform"`
	jwt.RegisteredClaims
}

func GenerateJwtToken(id int64, name, platform string) (string, error) {
	expDate := time.Now().Add(30 * time.Minute)

	if platform != PlatformWeb && platform != PlatformMobile {
		return "", errors.New("invalid platform for the token")
	}
	claims := &CustomClaims{
		UserID:    id,
		Name:      name,
		XPlatform: platform,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expDate),
			Subject:   fmt.Sprint(id),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims, nil)
	return token.SignedString(jwtKey)
}

func VerifyJWT(tokenStr string) (int64, string, string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, CustomClaims{}, func(w *jwt.Token) (any, error) {
		_, ok := w.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.New("unexpected signin method!")
		}
		return jwtKey, nil
	})
	if err != nil {
		return 0, "", "", fmt.Errorf("token parse failed, %v", err)
	}

	if !token.Valid {
		return 0, "", "", fmt.Errorf("invalid toekn")
	}
	claims, ok := token.Claims.(*CustomClaims)

	if !ok {
		return 0, "", "", fmt.Errorf("invalid claims error %v", err)
	}

	if claims.UserID == 0 || claims.Name == "" || (claims.XPlatform != PlatformWeb && claims.XPlatform != PlatformMobile) {
		return 0, "", "", fmt.Errorf("missing or invalid user claims error %v", err)
	}

	return claims.UserID, claims.Name, claims.XPlatform, nil
}
