package routes

import (
	"encoding/json"
	"net/http"

	"gotel/internals/models"
	"gotel/internals/utils"
)

func handleUserRegistration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		name     string "json:name"
		email    string "json:email"
		password string "json:password"
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.JSON(w, 500, false, "Invalid entered data, please try again", nil)
		return
	}

	if req.email == "" || req.name == "" || req.password == "" {
		utils.JSON(w, http.StatusBadRequest, false, "Invalid credentials", nil)
		return
	}

	if existingUser, _ := models.GetUserByEmail(req.email); existingUser != nil {
		utils.JSON(w, http.StatusConflict, false, "Email is already exists", nil)
		return
	}

	hashedPassword, err := utils.HashPassword(req.password)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "Signed up failed, please try again later", nil)
		return
	}

	user, err := models.CreateUser(req.name, req.email, hashedPassword)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "Could not create an user", nil)
		return
	}
	_ = user

	utils.JSON(w, http.StatusCreated, true, "User has been created successfully", nil)
}
