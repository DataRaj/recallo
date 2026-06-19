package utils

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Status  int    `json:"status"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func JSON(w http.ResponseWriter, status int, success bool, message string, data any) {
	resp := APIResponse{
		Status:  status,
		Success: success,
		Message: message,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status) // ← was hardcoded http.StatusOK — now uses the actual status

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Headers already sent; log is the only option here.
		http.Error(w, `{"status":500,"success":false,"message":"Internal server error"}`, http.StatusInternalServerError)
	}
}

