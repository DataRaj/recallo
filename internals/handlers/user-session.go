package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"gotel/internals/middleware"
	"gotel/internals/utils"
)

type RefreshSessionRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func HandleRefreshSession(w http.ResponseWriter, r *http.Request) {
	platform := strings.ToLower(strings.TrimSpace(r.Header.Get(middleware.CtxPlatform)))
	if platform != utils.PlatformWeb && platform != utils.PlatformMobile {
		utils.JSON(w, http.StatusBadRequest, false, "Invalid platform", nil)
		return
	}

	var req RefreshSessionRequest

	err := json.NewDecoder(r.Body).Decode(req)
	defer r.Body.Close()

	if err != nil {
		utils.JSON(w, http.StatusBadRequest, false, "Invalid request data", nil)
		return
	}

	if req.RefreshToken == "" {
		utils.JSON(w, http.StatusBadRequest, false, "refresh token is required", nil)
		return
	}

	user, err := utils.GetUserByRefreshToken(req.RefreshToken, platform)
	if err != nil || user == nil {
		utils.JSON(w, http.StatusUnauthorized, false, "invalid refresh token", nil)
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

	utils.JSON(w, http.StatusOK, true, "session refreshed successfully", map[string]any{
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}
