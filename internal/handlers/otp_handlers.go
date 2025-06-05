package handlers

import (
	"cource-api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// HandleVerifyOTP verifies the OTP for registration
func HandleVerifyOTP(otpRepo *repository.OTPRepository, userRepo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Email string `json:"email"`
			OTP   string `json:"otp"`
		}

		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate email
		if err := validateEmail(req.Email); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		// Get latest OTP
		otp, err := otpRepo.GetLatestOTP(c.Context(), req.Email, "registration")
		if err != nil {
			logrus.WithError(err).Error("Failed to get OTP")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify OTP")
		}

		if otp == nil {
			return fiber.NewError(fiber.StatusBadRequest, "No valid OTP found")
		}

		// Verify OTP
		if otp.Code != req.OTP {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid OTP")
		}

		// Mark OTP as used
		if err := otpRepo.MarkAsUsed(c.Context(), otp.ID); err != nil {
			logrus.WithError(err).Error("Failed to mark OTP as used")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify OTP")
		}

		// Get user by email
		user, err := userRepo.GetByEmail(c.Context(), req.Email)
		if err != nil {
			logrus.WithError(err).Error("Failed to get user")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify user")
		}

		if user == nil {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		// Update user verification status
		user.IsVerified = true
		if err := userRepo.Update(c.Context(), user); err != nil {
			logrus.WithError(err).Error("Failed to update user verification status")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify user")
		}

		return c.JSON(fiber.Map{
			"message": "Email verified successfully",
		})
	}
}
