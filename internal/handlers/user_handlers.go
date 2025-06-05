package handlers

import (
	"cource-api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
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
			Email    string `json:"email"`
			Password string `json:"password,omitempty"`
		}

		if err := c.BodyParser(&updateData); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Update user fields
		if updateData.Email != "" {
			user.Email = updateData.Email
		}

		if updateData.Password != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updateData.Password), bcrypt.DefaultCost)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to hash password")
			}
			user.PasswordHash = string(hashedPassword)
		}

		// Update in database
		if err := repo.Update(c.Context(), user); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update user")
		}

		return c.JSON(user)
	}
}
