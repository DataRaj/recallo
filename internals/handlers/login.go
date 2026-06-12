package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"recallo/internals/logger"
	"recallo/internals/middleware"
	"recallo/internals/models"
	"recallo/internals/utils"
)

type EmailLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func HandleEmailLogin(w http.ResponseWriter, r *http.Request) {
	platform := strings.ToLower(strings.TrimSpace(r.Header.Get(middleware.CtxPlatform)))

	if platform != utils.PlatformWeb && platform != utils.PlatformMobile {
		logger.App.Printf("[LOGIN] error=invalid_platform platform=%q remote=%s", platform, r.RemoteAddr)
		utils.JSON(w, http.StatusBadRequest, false, "Invalid platform", nil)
		return
	}

	var req EmailLoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	defer r.Body.Close()
	if err != nil {
		logger.App.Printf("[LOGIN] error=decode_failure platform=%s remote=%s err=%v", platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusBadRequest, false, "invalid requested data", nil)
		return
	}

	if req.Email == "" || req.Password == "" {
		logger.App.Printf("[LOGIN] error=missing_credentials platform=%s remote=%s email=%q", platform, r.RemoteAddr, req.Email)
		utils.JSON(w, http.StatusBadRequest, false, "email and password are required", nil)
		return
	}

	logger.App.Printf("[LOGIN] attempt email=%q platform=%s remote=%s", req.Email, platform, r.RemoteAddr)

	user, err := models.GetUserByEmail(req.Email)
	if err != nil || user == nil {
		logger.App.Printf("[LOGIN] error=user_not_found email=%q platform=%s remote=%s", req.Email, platform, r.RemoteAddr)
		utils.JSON(w, http.StatusUnauthorized, false, "invalid email or password", nil)
		return
	}

	if err := utils.CheckHashedPassword(req.Password, user.Password); err != nil {
		logger.App.Printf("[LOGIN] error=wrong_password user_id=%d email=%q platform=%s remote=%s", user.ID, req.Email, platform, r.RemoteAddr)
		utils.JSON(w, http.StatusUnauthorized, false, "invalid email or password", nil)
		return
	}

	accessToken, err := utils.GenerateJwtToken(user.ID, user.Name, platform)
	if err != nil {
		logger.App.Printf("[LOGIN] error=jwt_generation_failure user_id=%d platform=%s remote=%s err=%v", user.ID, platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to generate token", nil)
		return
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		logger.App.Printf("[LOGIN] error=refresh_token_generation_failure user_id=%d platform=%s remote=%s err=%v", user.ID, platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to generate refresh token", nil)
		return
	}

	err = utils.UpdateRefreshToken(user.ID, platform, refreshToken)
	if err != nil {
		logger.App.Printf("[LOGIN] error=refresh_token_save_failure user_id=%d platform=%s remote=%s err=%v", user.ID, platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to save refresh token", nil)
		return
	}

	logger.App.Printf("[LOGIN] success user_id=%d email=%q platform=%s remote=%s", user.ID, req.Email, platform, r.RemoteAddr)
	utils.JSON(w, http.StatusOK, true, "login successful", map[string]any{
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxUserID).(int64)
	if !ok {
		logger.App.Printf("[LOGOUT] error=missing_user_id_in_context remote=%s", r.RemoteAddr)
		utils.JSON(w, http.StatusUnauthorized, false, "Unauthorized user", nil)
		return
	}

	platform, ok := r.Context().Value(middleware.CtxPlatform).(string)
	if !ok {
		logger.App.Printf("[LOGOUT] error=missing_platform_in_context user_id=%d remote=%s", userID, r.RemoteAddr)
		utils.JSON(w, http.StatusBadRequest, false, "Invalid platform", nil)
		return
	}

	logger.App.Printf("[LOGOUT] attempt user_id=%d platform=%s remote=%s", userID, platform, r.RemoteAddr)

	err := utils.DeleteUserRefreshToken(userID, platform)
	if err != nil {
		logger.App.Printf("[LOGOUT] error=token_delete_failure user_id=%d platform=%s remote=%s err=%v", userID, platform, r.RemoteAddr, err)
		utils.JSON(w, http.StatusInternalServerError, false, "Something went wrong please retry", nil)
		return
	}

	logger.App.Printf("[LOGOUT] success user_id=%d platform=%s remote=%s", userID, platform, r.RemoteAddr)
	utils.JSON(w, http.StatusOK, true, "User logged out successfully", nil)
}
