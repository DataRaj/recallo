package routes

import (
	"net/http"

	"recallo/internals/handlers"
	"recallo/internals/middleware"
)

// RegisterRoutes builds and returns the root HTTP mux with all application
// routes registered. The mux is already wrapped with CORS middleware here so
// that every route benefits from it automatically.
func RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/healthcheck", Healthcheck)
	mux.HandleFunc("POST /api/v1/auth/register", handleUserRegistration)
	mux.HandleFunc("POST /api/v1/auth/login", handlers.HandleEmailLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh-session", handlers.HandleRefreshSession)

	// ── Protected (require valid JWT) ─────────────────────────────────────────
	mux.Handle("POST /api/v1/auth/logout",
		middleware.Authenticate(http.HandlerFunc(handlers.HandleLogout)))

	mux.Handle("GET /api/v1/auth/current-user",
		middleware.Authenticate(http.HandlerFunc(handlers.HandleGetCurrentUser)))

	mux.Handle("GET /api/v1/users/{id}", middleware.Authenticate(http.HandlerFunc(handlers.GetUserByID)))

	mux.Handle("GET /api/v1/private/{private}", middleware.Authenticate(http.HandlerFunc(handlers.HandleGetPrivate)))
	// Wrap entire mux with CORS so preflight OPTIONS requests are handled globally.

	mux.Handle("POST /api/v1/files/{private_id}", middleware.Authenticate(http.HandlerFunc(handlers.HandleFileUpload)))
	mux.Handle("GET /api/v1/files/", middleware.AuthenticateHandler(handlers.HandleGetFile()))
	return middleware.CORSMiddleware(mux)
}
