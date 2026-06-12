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

	// Build route handler (CORS already applied inside RegisterRoutes).
	routeHandler := routes.RegisterRoutes()

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
		log.Printf("[recallo] server starting on %s  [env=%s]", cfg.HTTPServer.Address, cfg.Env)
		log.Println("  GET  /api/v1/healthcheck")
		log.Println("  POST /api/v1/auth/register")
		log.Println("  POST /api/v1/auth/login")
		log.Println("  POST /api/v1/auth/refresh")
		log.Println("  POST /api/v1/auth/logout   [protected]")
		log.Println("  POST /api/v1/auth/refresh-session   [protected]")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[recallo] server error: %v", err)
		}
	}()

	sig := <-shutdownCh
	log.Printf("[recallo] signal received: %v — initiating graceful shutdown", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[recallo] shutdown error: %v", err)
	} else {
		log.Println("[recallo] server shut down cleanly")
	}
}
