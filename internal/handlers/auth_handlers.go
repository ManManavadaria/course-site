package handlers

import (
	"cource-api/internal/config"
	"cource-api/internal/middleware"
	"cource-api/internal/models"
	"cource-api/internal/repository"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// validateEmail checks if the email is valid
func validateEmail(email string) error {
	if len(email) == 0 {
		return errors.New("email is required")
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

// validatePassword checks if the password meets the requirements
func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*]`).MatchString(password)

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return errors.New("password must contain at least one uppercase letter, one lowercase letter, one number, and one special character")
	}
	return nil
}

// HandleRegister handles user registration
func HandleRegister(repo *repository.UserRepository, otpRepo *repository.OTPRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			logrus.WithError(err).Error("Failed to parse registration request body")
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate email
		if err := validateEmail(req.Email); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		// Validate password
		if err := validatePassword(req.Password); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		// Check if user already exists
		existingUser, err := repo.GetByEmail(c.Context(), req.Email)
		if err == nil && existingUser != nil {
			if !existingUser.IsVerified {
				otp, err := GenerateAndSaveOTP(c.Context(), otpRepo, req.Email, "registration")
				if err != nil {
					logrus.WithError(err).Error("Failed to generate OTP during registration")
					return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate verification code")
				}

				fmt.Println(otp)
				return c.JSON(fiber.Map{
					"message": "User already registered. Please verify your email with the OTP.",
				})
			}
			return fiber.NewError(fiber.StatusConflict, "User already exists")
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			logrus.WithError(err).Error("Failed to hash password during registration")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to process registration")
		}

		// Create user
		user := &models.User{
			Name:         req.Name,
			Email:        req.Email,
			PasswordHash: string(hashedPassword),
			Role:         "user",
			IsVerified:   false,
			Blocked:      false,
		}

		if err := repo.Create(c.Context(), user); err != nil {
			logrus.WithError(err).WithField("email", req.Email).Error("Failed to create user during registration")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create user")
		}

		// Generate and save OTP
		otp, err := GenerateAndSaveOTP(c.Context(), otpRepo, req.Email, "registration")
		if err != nil {
			logrus.WithError(err).Error("Failed to generate OTP during registration")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate verification code")
		}

		fmt.Println(otp)

		return c.JSON(fiber.Map{
			"message": "Registration successful. Please verify your email with the OTP.",
		})
	}
}

// HandleLogin handles user login
func HandleLogin(repo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			logrus.WithError(err).Error("Failed to parse login request body")
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate email
		if err := validateEmail(req.Email); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		if req.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Password is required")
		}

		// Get user by email
		user, err := repo.GetByEmail(c.Context(), req.Email)
		if err != nil {
			logrus.WithError(err).WithField("email", req.Email).Error("Failed to get user during login")
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid credentials")
		}

		if user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid credentials")
		}

		if !user.IsVerified {
			return fiber.NewError(fiber.StatusForbidden, "Account is not verified")
		}

		// Check if account is blocked
		if user.Blocked {
			return fiber.NewError(fiber.StatusForbidden, "Account is blocked")
		}

		// Verify password
		if !user.VerifyPassword(req.Password) {
			fmt.Println("Pass error")
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid credentials")
		}

		// Generate JWT token
		token, err := generateToken(user)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"user_id": user.ID,
				"email":   user.Email,
			}).Error("Failed to generate token during login")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate token")
		}

		return c.JSON(fiber.Map{
			"token": token,
			"user":  user,
		})
	}
}

// GetUserFromContext extracts user from context
func GetUserFromContext(c *fiber.Ctx) (*models.User, error) {
	claims, ok := c.Locals("user").(*middleware.Claims)
	if !ok {
		logrus.Error("Failed to get user claims from context")
		return nil, fiber.NewError(fiber.StatusUnauthorized, "User not found in context")
	}

	// Convert Claims to User
	user := &models.User{
		ID:    claims.UserID,
		Email: claims.Email,
		Role:  claims.Role,
	}

	return user, nil
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(c *fiber.Ctx) (string, error) {
	user, err := GetUserFromContext(c)
	if err != nil {
		return "", err
	}
	return user.ID.Hex(), nil
}

// generateToken generates a JWT token for the user
func generateToken(user *models.User) (string, error) {
	claims := &middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.AppConfig.JWTExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// HandleRequestPasswordReset handles password reset request
func HandleRequestPasswordReset(userRepo *repository.UserRepository, otpRepo *repository.OTPRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Email string `json:"email"`
		}

		if err := c.BodyParser(&req); err != nil {
			logrus.WithError(err).Error("Failed to parse password reset request body")
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate email
		if err := validateEmail(req.Email); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		// Check if user exists
		user, err := userRepo.GetByEmail(c.Context(), req.Email)
		if err != nil {
			logrus.WithError(err).WithField("email", req.Email).Error("Failed to get user during password reset request")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to process password reset request")
		}

		// If user exists, generate and save OTP
		if user != nil {
			otp, err := GenerateAndSaveOTP(c.Context(), otpRepo, req.Email, "reset")
			if err != nil {
				logrus.WithError(err).WithField("email", req.Email).Error("Failed to generate OTP for password reset")
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to process password reset request")
			}

			logrus.WithFields(logrus.Fields{
				"email": req.Email,
				"otp":   otp.Code,
			}).Info("Generated password reset OTP")
		}

		// Always return success to prevent email enumeration
		return c.JSON(fiber.Map{
			"message": "If your email is registered, you will receive a password reset code",
		})
	}
}

// HandleResetPassword handles password reset with OTP verification
func HandleResetPassword(userRepo *repository.UserRepository, otpRepo *repository.OTPRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Email       string `json:"email"`
			OTP         string `json:"otp"`
			NewPassword string `json:"new_password"`
		}

		if err := c.BodyParser(&req); err != nil {
			logrus.WithError(err).Error("Failed to parse password reset body")
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate email
		if err := validateEmail(req.Email); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		// Validate new password
		if err := validatePassword(req.NewPassword); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		// Get latest OTP
		otp, err := otpRepo.GetLatestOTP(c.Context(), req.Email, "reset")
		if err != nil {
			logrus.WithError(err).Error("Failed to get OTP")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify reset code")
		}

		if otp == nil {
			return fiber.NewError(fiber.StatusBadRequest, "No valid reset code found")
		}

		// Verify OTP
		if otp.Code != req.OTP {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid reset code")
		}

		// Mark OTP as used
		if err := otpRepo.MarkAsUsed(c.Context(), otp.ID); err != nil {
			logrus.WithError(err).Error("Failed to mark OTP as used")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify reset code")
		}

		// Get user
		user, err := userRepo.GetByEmail(c.Context(), req.Email)
		if err != nil {
			logrus.WithError(err).WithField("email", req.Email).Error("Failed to get user during password reset")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to reset password")
		}
		if user == nil {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		// Hash new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			logrus.WithError(err).Error("Failed to hash new password")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to reset password")
		}

		// Update user's password
		user.PasswordHash = string(hashedPassword)
		if err := userRepo.Update(c.Context(), user); err != nil {
			logrus.WithError(err).WithField("email", req.Email).Error("Failed to update user password")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to reset password")
		}

		return c.JSON(fiber.Map{
			"message": "Password has been reset successfully",
		})
	}
}
