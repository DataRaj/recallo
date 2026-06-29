package middleware

import (
	"context"
	"net/http"
	"strings"

	"recallo/internals/logger"
	"recallo/internals/utils"
)

const (
	CtxUserID          string = "userId"
	CtxUserDisplayName string = "name"
	CtxPlatform        string = "X-Platform"
	CtxAuthorization   string = "Authorization"
)

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		authHeader := strings.TrimSpace(r.Header.Get(CtxAuthorization))
		if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			log.Warn("authentication failed: missing or invalid authorization header")
			utils.JSON(w, http.StatusUnauthorized, false, "Unauthorized", nil)
			return
		}

		accessToken := strings.TrimSpace(authHeader[7:])

		userId, name, tokenPlatform, err := utils.VerifyJWT(accessToken)
		if err != nil {
			log.Warn("authentication failed: invalid jwt token", "error", err)
			utils.JSON(w, http.StatusUnauthorized, false, "Unauthorized", nil)
			return
		}

		platform := strings.ToLower(strings.TrimSpace(r.Header.Get(CtxPlatform)))
		if platform == "" {
			platform = tokenPlatform
		}

		if platform != utils.PlatformWeb && platform != utils.PlatformMobile {
			log.Warn("authentication failed: invalid platform header", "platform", platform)
			utils.JSON(w, http.StatusBadRequest, false, "Invalid platform", nil)
			return
		}

		if tokenPlatform != platform {
			log.Warn("authentication failed: platform mismatch", "token_platform", tokenPlatform, "request_platform", platform)
			utils.JSON(w, http.StatusUnauthorized, false, "Unauthorized", nil)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, CtxUserID, userId)
		ctx = context.WithValue(ctx, CtxUserDisplayName, name)
		ctx = context.WithValue(ctx, CtxPlatform, platform)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
