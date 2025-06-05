package handlers

import (
	"cource-api/internal/models"
	"cource-api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HandleCreateSubscription creates a new subscription
func HandleCreateSubscription(subRepo *repository.SubscriptionRepository, productRepo *repository.ProductRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request struct {
			ProductID       string `json:"product_id"`
			PaymentMethodID string `json:"payment_method_id"`
		}
		if err := c.BodyParser(&request); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		productID, err := primitive.ObjectIDFromHex(request.ProductID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid product ID")
		}

		product, err := productRepo.GetByID(c.Context(), productID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "Product not found")
		}

		userID := c.Locals("user_id").(primitive.ObjectID)
		subscription := &models.Subscription{
			UserID:          userID,
			ProductID:       productID,
			Status:          "active",
			Plan:            product.Type,
			Currency:        product.Currency,
			Amount:          product.Price,
			PaymentMethodID: request.PaymentMethodID,
			AutoRenew:       true,
		}

		if err := subRepo.Create(c.Context(), subscription); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create subscription")
		}

		return c.Status(fiber.StatusCreated).JSON(subscription)
	}
}

// HandleGetSubscription retrieves a subscription by ID
func HandleGetSubscription(repo *repository.SubscriptionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid subscription ID")
		}

		subscription, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "Subscription not found")
		}

		// Verify ownership
		userID := c.Locals("user_id").(primitive.ObjectID)
		if subscription.UserID != userID {
			return fiber.NewError(fiber.StatusForbidden, "Not authorized to access this subscription")
		}

		return c.JSON(subscription)
	}
}

// HandleListSubscriptions returns a paginated list of subscriptions for the current user
func HandleListSubscriptions(repo *repository.SubscriptionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page := c.QueryInt("page", 1)
		limit := c.QueryInt("limit", 10)
		userID := c.Locals("user_id").(primitive.ObjectID)

		subscriptions, total, err := repo.ListByUser(c.Context(), userID, int64(page), int64(limit))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to list subscriptions")
		}

		return c.JSON(fiber.Map{
			"subscriptions": subscriptions,
			"total":         total,
			"page":          page,
			"limit":         limit,
		})
	}
}

// HandleCancelSubscription cancels a subscription
func HandleCancelSubscription(repo *repository.SubscriptionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid subscription ID")
		}

		subscription, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "Subscription not found")
		}

		// Verify ownership
		userID := c.Locals("user_id").(primitive.ObjectID)
		if subscription.UserID != userID {
			return fiber.NewError(fiber.StatusForbidden, "Not authorized to cancel this subscription")
		}

		subscription.Status = "canceled"
		subscription.CancelAtPeriodEnd = true
		if err := repo.Update(c.Context(), subscription); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to cancel subscription")
		}

		return c.JSON(subscription)
	}
}

// HandleUpdatePaymentMethod updates the payment method for a subscription
func HandleUpdatePaymentMethod(repo *repository.SubscriptionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid subscription ID")
		}

		var request struct {
			PaymentMethodID string `json:"payment_method_id"`
		}
		if err := c.BodyParser(&request); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		subscription, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "Subscription not found")
		}

		// Verify ownership
		userID := c.Locals("user_id").(primitive.ObjectID)
		if subscription.UserID != userID {
			return fiber.NewError(fiber.StatusForbidden, "Not authorized to update this subscription")
		}

		updates := map[string]interface{}{
			"payment_method_id": request.PaymentMethodID,
		}
		if err := repo.UpdatePaymentInfo(c.Context(), objectID, updates); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update payment method")
		}

		return c.SendStatus(fiber.StatusOK)
	}
}

// HandleReactivateSubscription reactivates a canceled subscription
func HandleReactivateSubscription(repo *repository.SubscriptionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid subscription ID")
		}

		subscription, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "Subscription not found")
		}

		// Verify ownership
		userID := c.Locals("user_id").(primitive.ObjectID)
		if subscription.UserID != userID {
			return fiber.NewError(fiber.StatusForbidden, "Not authorized to reactivate this subscription")
		}

		subscription.Status = "active"
		subscription.CancelAtPeriodEnd = false
		if err := repo.Update(c.Context(), subscription); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to reactivate subscription")
		}

		return c.JSON(subscription)
	}
}
