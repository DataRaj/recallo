package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"recallo/internals/logger"
	"recallo/internals/middleware"
	"recallo/internals/utils"
)

type RefreshSessionRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func HandleRefreshSession(w http.ResponseWriter, r *http.Request) {
	platform := strings.ToLower(strings.TrimSpace(r.Header.Get(middleware.CtxPlatform)))
	if platform != utils.PlatformWeb && platform != utils.PlatformMobile {
		logger.App.Printf("[REFRESH] error=invalid_platform platform=%q remote=%s", platform, r.RemoteAddr)
		utils.JSON(w, http.StatusBadRequest, false, "Invalid platform", nil)
		return
	}

	var req RefreshSessionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	defer r.Body.Close()
	if err != nil {
		logger.App.Printf("[REFRESH] error=decode_failure platform=%s remote=%s err=%v", platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusBadRequest, false, "Invalid request data", nil)
		return
	}

	if req.RefreshToken == "" {
		logger.App.Printf("[REFRESH] error=missing_refresh_token platform=%s remote=%s", platform, r.RemoteAddr)
		utils.JSON(w, http.StatusBadRequest, false, "refresh token is required", nil)
		return
	}

	logger.App.Printf("[REFRESH] attempt platform=%s remote=%s", platform, r.RemoteAddr)

	user, err := utils.GetUserByRefreshToken(req.RefreshToken, platform)
	if err != nil || user == nil {
		logger.App.Printf("[REFRESH] error=invalid_or_expired_token platform=%s remote=%s err=%v", platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusUnauthorized, false, "invalid refresh token", nil)
		return
	}

	accessToken, err := utils.GenerateJwtToken(user.ID, user.Name, platform)
	if err != nil {
		logger.App.Printf("[REFRESH] error=jwt_generation_failure user_id=%d platform=%s remote=%s err=%v", user.ID, platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to generate token", nil)
		return
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		logger.App.Printf("[REFRESH] error=refresh_token_generation_failure user_id=%d platform=%s remote=%s err=%v", user.ID, platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to generate refresh token", nil)
		return
	}

	err = utils.UpdateRefreshToken(user.ID, platform, refreshToken)
	if err != nil {
		logger.App.Printf("[REFRESH] error=refresh_token_save_failure user_id=%d platform=%s remote=%s err=%v", user.ID, platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to save refresh token", nil)
		return
	}

	logger.App.Printf("[REFRESH] success user_id=%d platform=%s remote=%s", user.ID, platform, r.RemoteAddr)
	utils.JSON(w, http.StatusOK, true, "session refreshed successfully", map[string]any{
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	platform := strings.ToLower(strings.TrimSpace(r.Header.Get(middleware.CtxPlatform)))
	if platform != utils.PlatformWeb && platform != utils.PlatformMobile {
		logger.App.Printf("[REFRESH] error=invalid_platform platform=%q remote=%s", platform, r.RemoteAddr)
		utils.JSON(w, http.StatusBadRequest, false, "Invalid platform", nil)
		return
	}

	var req RefreshSessionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	defer r.Body.Close()
	if err != nil {
		logger.App.Printf("[REFRESH] error=decode_failure platform=%s remote=%s err=%v", platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusBadRequest, false, "Invalid request data", nil)
		return
	}

	if req.RefreshToken == "" {
		logger.App.Printf("[REFRESH] error=missing_refresh_token platform=%s remote=%s", platform, r.RemoteAddr)
		utils.JSON(w, http.StatusBadRequest, false, "refresh token is required", nil)
		return
	}

	logger.App.Printf("[REFRESH] attempt platform=%s remote=%s", platform, r.RemoteAddr)

	existingUser, err := utils.GetUserByRefreshToken(req.RefreshToken, platform)
	if err != nil || existingUser == nil {
		logger.App.Printf("[REFRESH] error=invalid_or_expired_token platform=%s remote=%s err=%v", platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusUnauthorized, false, "invalid refresh token", nil)
		return
	}

	utils.JSON(w, http.StatusOK, true, "Current user is fetched successfully", map[string]any{
		"user": existingUser,
	})
}
