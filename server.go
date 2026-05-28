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

// const dbName = "gotel-reservation"
// const userColl = "user"
//

var config = fiber.Config{
	ErrorHandler: func(c fiber.Ctx, err error) error {
		// Define your error response structure
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	},
}

func main() {
	client, err := mongo.Connect(options.Client().ApplyURI(dburi))
	if err != nil {
		panic(err)
	}

	// coll := client.Database(dbName).Collection(userColl)
	//
	// user := types.User{
	// 	FirstName: "Dat",
	// 	LastName:  "Das",
	// }
	//
	// ctx := context.Background()
	// res, err := coll.InsertOne(ctx, user)
	//
	// if err != nil {
	// 	fmt.Println(err)
	// }
	//
	// fmt.Println(res)

	userStore := collections.NewMongoUserStore(client)
	userHandler := handlers.NewUserHandler(userStore)

	fmt.Println("Connected to MongoDB")
	listenAddr := flag.String("listenAddr", ":5000", "The listen address of the API server")
	app := fiber.New(config)

	apiv1 := app.Group("api/v1")

	apiv1.Get("/user", userHandler.HandleGetUser)

	apiv1.Get("/users", userHandler.HandleGetUsers)
	apiv1.Get("/user/:id", userHandler.HandleGetUser)

	app.Listen(*listenAddr)
}
