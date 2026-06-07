package routes

import (
	"net/http"

	"gotel/internals/utils"
)

func Healthcheck(w http.ResponseWriter, r *http.Request) {
	utils.JSON(w, http.StatusOK, true, "API is running okay", nil)
}
