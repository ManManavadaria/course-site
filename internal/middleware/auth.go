package middleware

import (
	"cource-api/internal/config"
	"cource-api/internal/models"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Claims represents the JWT claims
type Claims struct {
	UserID primitive.ObjectID `json:"user_id"`
	Email  string             `json:"email"`
	Role   string             `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token
func GenerateToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.AppConfig.JWTExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// AuthMiddleware handles JWT authentication
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Authorization header is required")
		}

		// Extract token from Bearer
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		// Parse and validate token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.AppConfig.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid or expired token")
		}

		// Set user info in context
		c.Locals("user", claims)
		return c.Next()
	}
}

// RequireRole middleware ensures the user has the required role
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := c.Locals("user").(*Claims)

		for _, role := range roles {
			if user.Role == role {
				return c.Next()
			}
		}

		return fiber.NewError(fiber.StatusForbidden, "Insufficient permissions")
	}
}

// RequireSubscription middleware ensures the user has an active subscription
func RequireSubscription() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := c.Locals("user").(*Claims)

		// TODO: Check user's subscription status from database
		// For now, we'll just check if the user is not blocked
		if user.Role == "admin" {
			return c.Next()
		}

		// TODO: Implement subscription check
		// This is a placeholder - you'll need to implement the actual subscription check
		return c.Next()
	}
}
