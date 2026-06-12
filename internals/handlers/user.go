package handlers

import (
	"net/http"
	"strconv"

	"recallo/internals/models"
	"recallo/internals/utils"
)

func GetUserByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	parsedId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.JSON(w, http.StatusBadRequest, false, "Failed to parse the id", nil)
		return
	}

	existingUser, err := models.GetUserByID(parsedId)
	if err != nil {
		utils.JSON(w, http.StatusNotFound, false, "User not found", nil)
		return
	}

	utils.JSON(w, http.StatusOK, true, "User found Successfully", existingUser)
}
