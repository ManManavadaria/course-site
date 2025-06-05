package handlers

import (
	"cource-api/internal/models"
	"cource-api/internal/repository"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HandleListUsers lists all users (admin only)
func HandleListUsers(repo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get pagination parameters
		page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
		limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)

		// Get users with pagination
		users, total, err := repo.List(c.Context(), page, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to list users")
		}

		return c.JSON(fiber.Map{
			"users": users,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	}
}

// HandleUpdateUser updates a user (admin only)
func HandleUpdateUser(repo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user ID from params
		userID := c.Params("id")
		if userID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "User ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID format")
		}

		// Get existing user
		user, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get user")
		}
		if user == nil {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		// Parse update data
		var updateData models.User

		if err := c.BodyParser(&updateData); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Update user fields
		if updateData.Email != "" {
			user.Email = updateData.Email
		}
		if updateData.Name != "" {
			user.Name = updateData.Name
		}
		if updateData.Role != "" {
			user.Role = updateData.Role
		}
		user.IsVerified = updateData.IsVerified

		// Update user
		if err := repo.Update(c.Context(), user); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update user")
		}
		return c.JSON(user)
	}
}

// HandleDeleteUser deletes a user (admin only)
func HandleDeleteUser(repo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user ID from params
		userID := c.Params("id")
		if userID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "User ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID format")
		}

		// Delete user
		if err := repo.Delete(c.Context(), objectID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete user")
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// HandleUpdateRegionalPricing updates pricing for a specific region (admin only)
func HandleUpdateRegionalPricing(repo *repository.PaymentRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get region code from params
		regionCode := c.Params("region")
		if regionCode == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Region code is required")
		}

		// Parse pricing data
		var pricing models.RegionalPricing
		if err := c.BodyParser(&pricing); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate required fields
		if pricing.Currency == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Currency is required")
		}
		if pricing.MonthlyPrice <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Monthly price must be greater than 0")
		}
		if pricing.YearlyPrice <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Yearly price must be greater than 0")
		}

		// Set region code
		pricing.RegionCode = regionCode

		// Update pricing
		if err := repo.UpdateRegionalPricing(c.Context(), &pricing); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update regional pricing")
		}

		return c.JSON(pricing)
	}
}
