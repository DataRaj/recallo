package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gotel/db"
	"gotel/internals/configs"
	"gotel/internals/middleware"
	"gotel/internals/routes"
)

func main() {
	cfg := configs.LoadConfig()

	// Initialize PostgreSQL connection pool.
	if err := db.InitDB(cfg.DatabaseURL, db.DefaultConfig()); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.CloseDBConnection()

	// mux := http.NewServeMux()
	//

	mux := routes.RegisterRoutes()

	// Wrap the mux with the logging middleware
	handler := middleware.Loggingmiddleware(mux)

	// TODO: Register your routes here.
	// example: mux.Handle("/api/v1/", handlers.NewRouter(db.GetDB()))

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
		log.Printf("Server is running on %s [env=%s]", cfg.HTTPServer.Address, cfg.Env)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	sig := <-shutdownCh
	log.Printf("Shutdown signal received: %v — initiating graceful shutdown", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	} else {
		log.Println("Server shut down cleanly")
	}
}
