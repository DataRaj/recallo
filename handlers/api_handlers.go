package handlers

import (
	"gotel/types"

	"github.com/gofiber/fiber/v3"
)

func HandleUser(c fiber.Ctx) error {
	// john := c.`json:"john going to be missed here!"`
	u := types.User{
		// ID:        "23423",
		FirstName: "ladis",
		LastName:  "washroom",
	}
	return c.JSON(u)

}

func GetUserById(c fiber.Ctx) error {
	return c.JSON(map[string]string{"userId": "23422"})
}
