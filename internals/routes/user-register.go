package routes

import (
	"encoding/json"
	"net/http"

	"recallo/internals/models"
	"recallo/internals/utils"
)

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleUserRegistration(w http.ResponseWriter, r *http.Request) {
	var req registerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}
	defer r.Body.Close()

	if req.Email == "" || req.Name == "" || req.Password == "" {
		utils.JSON(w, http.StatusBadRequest, false, "Invalid credentials", nil)
		return
	}

	if existingUser, _ := models.GetUserByEmail(req.Email); existingUser != nil {
		utils.JSON(w, http.StatusConflict, false, "Email is already exists", nil)
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "Signed up failed, please try again later", nil)
		return
	}

	user, err := models.CreateUser(req.Name, req.Email, hashedPassword)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "Could not create user", nil)
		return
	}

	utils.JSON(w, http.StatusCreated, true, "Account created successfully", map[string]any{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	})
}
