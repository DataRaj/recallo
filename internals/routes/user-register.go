package routes

import (
	"encoding/json"
	"net/http"

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
	}
}
