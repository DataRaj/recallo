package main

import (
	"flag"
	"gotel/handlers"

	"github.com/gofiber/fiber/v3"
)

func main() {
	listenAddr := flag.String("listenAddr", ":5000", "The listen address of the API server")
	app := fiber.New()

	apiv1 := app.Group("api/v1")

	app.Get("/", handleGetHi)
	apiv1.Get("/user", handlers.HandleUser)

	app.Listen(*listenAddr)
}

func handleGetHi(c fiber.Ctx) error {
	return c.JSON(map[string]string{"welcome": "Hello, Welcome to the Industry right here!"})

}

func handleUser(c fiber.Ctx) error {
	return c.JSON(map[string]string{"user": "John Doe"})
}
