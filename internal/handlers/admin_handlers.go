package handlers

import (
	"cource-api/internal/models"
	"cource-api/internal/repository"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// HandleListUsers lists all users with pagination and filtering
func HandleListUsers(repo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get pagination parameters
		page, err := strconv.ParseInt(c.Query("page", "1"), 10, 64)
		if err != nil || page < 1 {
			page = 1
		}
		limit, err := strconv.ParseInt(c.Query("limit", "10"), 10, 64)
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}

		// Get filter parameters
		role := c.Query("role")
		isVerified := c.Query("is_verified")
		isBlocked := c.Query("is_blocked")
		search := c.Query("search")

		// Build filter
		filter := make(map[string]interface{})
		if role != "" {
			filter["role"] = role
		}
		if isVerified != "" {
			verified, err := strconv.ParseBool(isVerified)
			if err == nil {
				filter["is_verified"] = verified
			}
		}
		if isBlocked != "" {
			blocked, err := strconv.ParseBool(isBlocked)
			if err == nil {
				filter["blocked"] = blocked
			}
		}
		if search != "" {
			filter["$or"] = []map[string]interface{}{
				{"name": map[string]string{"$regex": search, "$options": "i"}},
				{"email": map[string]string{"$regex": search, "$options": "i"}},
			}
		}

		// Get users
		users, total, err := repo.ListWithFilter(c.Context(), filter, page, limit)
		if err != nil {
			logrus.WithError(err).Error("Failed to list users")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve users")
		}

		return c.JSON(fiber.Map{
			"users": users,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	}
}

// HandleUpdateUser updates a user's information
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
			logrus.WithError(err).WithField("user_id", userID).Error("Invalid user ID format")
			return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID format")
		}

		// Get existing user
		user, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			logrus.WithError(err).WithField("user_id", userID).Error("Failed to get user")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve user")
		}
		if user == nil {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		// Parse update data
		var updateData struct {
			Name        string `json:"name"`
			Email       string `json:"email"`
			Role        string `json:"role"`
			IsVerified  bool   `json:"is_verified"`
			Blocked     bool   `json:"blocked"`
			NewPassword string `json:"new_password"`
		}

		if err := c.BodyParser(&updateData); err != nil {
			logrus.WithError(err).Error("Failed to parse update request body")
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate role if provided
		if updateData.Role != "" && updateData.Role != "user" && updateData.Role != "admin" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid role")
		}

		// Update user fields
		if updateData.Name != "" {
			user.Name = updateData.Name
		}
		if updateData.Email != "" {
			// Check if email is already taken
			existingUser, err := repo.GetByEmail(c.Context(), updateData.Email)
			if err != nil {
				logrus.WithError(err).Error("Failed to check email availability")
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify email")
			}
			if existingUser != nil && existingUser.ID != user.ID {
				return fiber.NewError(fiber.StatusConflict, "Email already in use")
			}
			user.Email = updateData.Email
		}
		if updateData.Role != "" {
			user.Role = updateData.Role
		}
		user.IsVerified = updateData.IsVerified
		user.Blocked = updateData.Blocked

		// Update password if provided
		if updateData.NewPassword != "" {
			if err := validatePassword(updateData.NewPassword); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updateData.NewPassword), bcrypt.DefaultCost)
			if err != nil {
				logrus.WithError(err).Error("Failed to hash new password")
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update password")
			}
			user.PasswordHash = string(hashedPassword)
		}

		// Save updated user
		if err := repo.Update(c.Context(), user); err != nil {
			logrus.WithError(err).WithField("user_id", userID).Error("Failed to update user")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update user")
		}

		return c.JSON(user)
	}
}

// HandleDeleteUser deletes a user
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
			logrus.WithError(err).WithField("user_id", userID).Error("Invalid user ID format")
			return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID format")
		}

		// Get existing user
		user, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			logrus.WithError(err).WithField("user_id", userID).Error("Failed to get user")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve user")
		}
		if user == nil {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		// Delete user
		if err := repo.Delete(c.Context(), objectID); err != nil {
			logrus.WithError(err).WithField("user_id", userID).Error("Failed to delete user")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete user")
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// HandleGetUserStats gets user statistics
func HandleGetUserStats(repo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		stats, err := repo.GetUserStats(c.Context())
		if err != nil {
			logrus.WithError(err).Error("Failed to get user statistics")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve user statistics")
		}

		return c.JSON(stats)
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
