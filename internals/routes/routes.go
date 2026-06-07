package routes

import "net/http"

func RegisterRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/healthcheck", Healthcheck)
	mux.HandleFunc("POST /api/v1/user-register", handleUserRegistration)

	return mux
}
