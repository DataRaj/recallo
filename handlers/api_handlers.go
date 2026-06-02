package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"gotel/db/collections"

	"github.com/go-chi/chi/v5"
)

// writeJSON encodes v as JSON into w, setting the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError encodes a standard error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"success": false,
		"message": msg,
	})
}

// UserHandler handles HTTP requests related to users.
type UserHandler struct {
	userStore collections.UserStore
}

// NewUserHandler constructs a UserHandler with the provided store.
func NewUserHandler(userStore collections.UserStore) *UserHandler {
	return &UserHandler{userStore: userStore}
}

// HandleGetUser fetches a single user by their MongoDB ObjectID.
//
//	GET /api/v1/users/{id}
func (h *UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing user id")
		return
	}

	user, err := h.userStore.GetUserByID(context.Background(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// HandleGetUsers returns all users in the collection.
//
//	GET /api/v1/users
func (h *UserHandler) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userStore.GetUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve users")
		return
	}

	writeJSON(w, http.StatusOK, users)
}
