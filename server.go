package main

import (
	"flag"
	"fmt"
	"gotel/db/collections"
	"gotel/handlers"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const dburi = "mongodb://localhost:27017"

func main() {
	client, err := mongo.Connect(options.Client().ApplyURI(dburi))

	if err != nil {
		panic(err)
	}

	userStore := collections.NewMongoUserStore(client)
	userHandler := handlers.NewUserHandler(userStore)

	fmt.Println("Connected to MongoDB")
	listenAddr := flag.String("listenAddr", ":5000", "The listen address of the API server")
	app := fiber.New()

	apiv1 := app.Group("api/v1")

	app.Get("/", handleGetHi)
	apiv1.Get("/user/:id", userHandler.HandleGetUser)

	app.Listen(*listenAddr)
}

func handleGetHi(c fiber.Ctx) error {
	return c.JSON(map[string]string{"welco": "Hello, Welcome to the Industry right here!"})
}
