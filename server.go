package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"gotel/db/collections"
	"gotel/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const dbURI = "mongodb://localhost:27017"

func main() {
	listenAddr := flag.String("listenAddr", ":5000", "The listen address of the API server")
	flag.Parse()

	client, err := mongo.Connect(options.Client().ApplyURI(dbURI))
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB")

	userStore := collections.NewMongoUserStore(client)
	userHandler := handlers.NewUserHandler(userStore)

	r := chi.NewRouter()

	// Global middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/users", userHandler.HandleGetUsers)
		r.Get("/users/{id}", userHandler.HandleGetUser)
	})

	// 404 fallback
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "route not found",
		})
	})

	log.Printf("Server listening on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
