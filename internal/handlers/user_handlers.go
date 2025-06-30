package handlers

import (
	"cource-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

var userRepo *repository.UserRepository

// SetUserRepository sets the user repository instance
func SetUserRepository(repo *repository.UserRepository) {
	userRepo = repo
}

// HandleGetCurrentUser returns the current user's information
func HandleGetCurrentUser(repo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		user, err = repo.GetByEmail(c.Context(), user.Email)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get user")
		}

		if user == nil {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		return c.JSON(user)
	}
}

// HandleUpdateCurrentUser updates the current user's information
func HandleUpdateCurrentUser(repo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		var updateData struct {
			Name string `json:"name"`
		}

		if err := c.BodyParser(&updateData); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		if updateData.Name == "" || len(updateData.Name) > 50 {
			return fiber.NewError(fiber.StatusForbidden, "Invalid input")
		}

		user.Name = updateData.Name

		// Update in database
		if err := repo.Update(c.Context(), user); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update user")
		}

		return c.JSON(user)
	}
}
