package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gotel/internals/configs"
)

func main() {
	cfg := configs.LoadConfig()
	mux := http.NewServeMux()

	port := cfg.HTTPServer
	server := &http.Server{
		Addr:    port.Address,
		Handler: mux,
	}

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("%s ... server is running", port.Address)
		err := server.ListenAndServe()

		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)

	if err != nil {
		log.Printf("Server Shutdown Failed: %v", err)
	} else {
		log.Println("Server has Shutdown successfully")
	}

	log.Println("server exited cleanly without any interruption!")
}
