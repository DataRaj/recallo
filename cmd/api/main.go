package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"recallo/db"
	"recallo/internals/configs"
	"recallo/internals/logger"
	"recallo/internals/middleware"
	"recallo/internals/realtime"
	"recallo/internals/routes"
	"recallo/internals/utils"
)

func main() {
	cfg := configs.LoadConfig()

	// Initialise shared logger — writes to stdout AND logs/app.log.
	closeLog, err := logger.Init()
	if err != nil {
		log.Fatalf("[startup] failed to initialise logger: %v", err)
	}
	defer closeLog()

	logger.App.Printf("[startup] logger initialised — output also tailing logs/app.log")

	// Initialise JWT signing key so all handlers can sign/verify tokens.
	utils.InitJWT(cfg.JWTSecretKey)

	// Initialise PostgreSQL connection pool and run schema migrations.
	if err := db.InitDB(cfg.DatabaseURL, db.DefaultConfig()); err != nil {
		log.Fatalf("[startup] failed to initialise database: %v", err)
	}
	defer db.CloseDBConnection()

	hub := realtime.NewHub()
	defer hub.Shutdown()

	// Build route handler (CORS already applied inside RegisterRoutes).
	routeHandler := routes.RegisterRoutes(hub)

	// Apply logging middleware on top of the route handler.
	handler := middleware.Loggingmiddleware(routeHandler)

	server := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.App.Printf("[startup] server listening on %s  env=%s", cfg.HTTPServer.Address, cfg.Env)
		logger.App.Printf("[startup] registered routes:")
		logger.App.Printf("  GET  /api/v1/healthcheck")
		logger.App.Printf("  POST /api/v1/auth/register")
		logger.App.Printf("  POST /api/v1/auth/login")
		logger.App.Printf("  POST /api/v1/auth/refresh-session")
		logger.App.Printf("  POST /api/v1/auth/logout              [protected]")
		logger.App.Printf("  GET  /api/v1/auth/current-user        [protected]")
		logger.App.Printf("  GET  /api/v1/users/{id}               [protected]")
		logger.App.Printf("  GET  /api/v1/conversations            [protected]")
		logger.App.Printf("  POST /api/v1/conversation/private/create             [protected]")
		logger.App.Printf("  GET  /api/v1/conversation/private/{private_id}       [protected]")
		logger.App.Printf("  GET  /api/v1/conversation/private/{private_id}/messages [protected]")
		logger.App.Printf("  POST /api/v1/files/{private_id}       [protected]")
		logger.App.Printf("  GET  /api/v1/files/                   [protected]")
		logger.App.Printf("  GET  /api/v1/ws                       [websocket]")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.App.Printf("[server] fatal err=%v", err)
		}
	}()

	sig := <-shutdownCh
	logger.App.Printf("[server] signal received: %v — initiating graceful shutdown", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.App.Printf("[server] shutdown error: %v", err)
	} else {
		logger.App.Printf("[server] shut down cleanly")
	}
}
