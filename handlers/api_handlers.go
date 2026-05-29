package handlers

import (
	"context"

	"gotel/db/collections"

	"github.com/gofiber/fiber/v3"
)

type UserHandler struct {
	userStore collections.UserStore
}

func NewUserHandler(userStore collections.UserStore) *UserHandler {
	// john := c.`json:"john going to be missed here!"`
	return &UserHandler{
		userStore: userStore,
	}
}

func (h *UserHandler) HandleGetUser(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	c := context.Background()
	user, err := h.userStore.GetUserByID(c, id)
	if err != nil {
		return err
	}
	return ctx.JSON(user)
}

// created new handle user here
func (h *UserHandler) HandleGetUsers(ctx fiber.Ctx) error {
	users, error := h.userStore.GetUsers(ctx.Context())
	if error != nil {
		return nil
	}
	return ctx.JSON(users)
}
