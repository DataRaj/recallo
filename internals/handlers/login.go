package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

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
		utils.JSON(w, http.StatusBadRequest, false, "Invalid platform", nil)
		return
	}

	var req EmailLoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	defer r.Body.Close()
	if err != nil {
		utils.JSON(w, http.StatusBadRequest, false, "invalid requested data", nil)
		return
	}

	if req.Email == "" || req.Password == "" {
		utils.JSON(w, http.StatusBadRequest, false, "email and password are required", nil)
		return
	}

	user, err := models.GetUserByEmail(req.Email)
	if err != nil || user == nil {
		utils.JSON(w, http.StatusUnauthorized, false, "invalid email or password", nil)
		return
	}

	if err := utils.CheckHashedPassword(req.Password, user.Password); err != nil {
		utils.JSON(w, http.StatusUnauthorized, false, "invalid email or password", nil)
		return
	}

	accessToken, err := utils.GenerateJwtToken(user.ID, user.Name, platform)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "failed to generate token", nil)
		return
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "failed to generate refresh token", nil)
		return
	}

	err = utils.UpdateRefreshToken(user.ID, platform, refreshToken)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "failed to save refresh token", nil)
		return
	}

	utils.JSON(w, http.StatusOK, true, "login successful", map[string]any{
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxUserID).(int64)

	if !ok {
		utils.JSON(w, http.StatusUnauthorized, false, "Unauthorized user", nil)
		return
	}

	platform, ok := r.Context().Value(middleware.CtxPlatform).(string)

	if !ok {
		utils.JSON(w, http.StatusBadRequest, false, "Invalid platform", nil)
		return
	}

	err := utils.DeleteUserRefreshToken(userID, platform)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "Something went wrong please retry", nil)
		return
	}

	utils.JSON(w, http.StatusOK, true, "User logged out successfully", nil)
}
