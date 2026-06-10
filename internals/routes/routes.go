package routes

import (
	"net/http"

	"gotel/internals/handlers"
)

func RegisterRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/healthcheck", Healthcheck)
	mux.HandleFunc("POST /api/v1/auth/register", handleUserRegistration)
	mux.HandleFunc("POST /api/v1/auth/login", handlers.HandleEmailLogin)

	return mux
}
