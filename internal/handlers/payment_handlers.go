package handlers

import (
	"cource-api/internal/config"
	"cource-api/internal/models"
	"cource-api/internal/repository"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/webhook"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HandleCreatePayment creates a new payment session
func HandleCreatePayment(repo *repository.PaymentRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get current user
		user, err := GetUserFromContext(c)
		if err != nil {
			logrus.WithError(err).Error("Failed to get user from context")
			return fiber.NewError(fiber.StatusUnauthorized, "Authentication required")
		}

		// Parse request body
		var req struct {
			PlanType string `json:"plan_type"`
			Region   string `json:"region"`
		}

		if err := c.BodyParser(&req); err != nil {
			logrus.WithError(err).Error("Failed to parse payment request body")
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate request
		if req.PlanType == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Plan type is required")
		}
		if req.Region == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Region is required")
		}

		// Get pricing for region
		pricing, err := repo.GetRegionalPricing(c.Context(), req.Region)
		if err != nil {
			logrus.WithError(err).WithField("region", req.Region).Error("Failed to get regional pricing")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get pricing information")
		}
		if pricing == nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid region or pricing not found")
		}

		// Set Stripe API key
		if config.AppConfig.StripeKey == "" {
			logrus.Error("Stripe API key is not configured")
			return fiber.NewError(fiber.StatusInternalServerError, "Payment system is not properly configured")
		}
		stripe.Key = config.AppConfig.StripeKey

		// Create or get Stripe customer
		var stripeCustomer *stripe.Customer
		listParams := &stripe.CustomerListParams{
			Email: stripe.String(user.Email),
		}
		iter := customer.List(listParams)
		if iter.Next() {
			if cust, ok := iter.Current().(*stripe.Customer); ok {
				stripeCustomer = cust
			}
		} else {
			custParams := &stripe.CustomerParams{
				Email: stripe.String(user.Email),
				Metadata: map[string]string{
					"user_id": user.ID.Hex(),
				},
			}
			stripeCustomer, err = customer.New(custParams)
			if err != nil {
				logrus.WithError(err).WithField("email", user.Email).Error("Failed to create Stripe customer")
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create customer account")
			}
		}

		// Determine price based on plan type
		var price int64
		if req.PlanType == "yearly" {
			price = int64(pricing.YearlyPrice)
		} else if req.PlanType == "monthly" {
			price = int64(pricing.MonthlyPrice)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid plan type")
		}

		// Create checkout session
		sessionParams := &stripe.CheckoutSessionParams{
			Customer: stripe.String(stripeCustomer.ID),
			PaymentMethodTypes: stripe.StringSlice([]string{
				"card",
			}),
			Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
						Currency: stripe.String(pricing.Currency),
						ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
							Name: stripe.String("Course Subscription"),
						},
						UnitAmount: stripe.Int64(price),
						Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
							Interval: stripe.String(req.PlanType),
						},
					},
					Quantity: stripe.Int64(1),
				},
			},
			SuccessURL: stripe.String("http://localhost:3000/success?session_id={CHECKOUT_SESSION_ID}"),
			CancelURL:  stripe.String("http://localhost:3000/cancel"),
		}

		session, err := session.New(sessionParams)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"user_id":   user.ID,
				"plan_type": req.PlanType,
				"region":    req.Region,
			}).Error("Failed to create checkout session")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create payment session")
		}

		return c.JSON(fiber.Map{
			"session_id": session.ID,
			"url":        session.URL,
		})
	}
}

// HandleGetPayment gets a payment by ID
func HandleGetPayment(repo *repository.PaymentRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get payment ID from params
		paymentID := c.Params("id")
		if paymentID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Payment ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(paymentID)
		if err != nil {
			logrus.WithError(err).WithField("payment_id", paymentID).Error("Invalid payment ID format")
			return fiber.NewError(fiber.StatusBadRequest, "Invalid payment ID format")
		}

		// Get payment
		payment, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			logrus.WithError(err).WithField("payment_id", paymentID).Error("Failed to get payment")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve payment information")
		}
		if payment == nil {
			return fiber.NewError(fiber.StatusNotFound, "Payment not found")
		}

		// Verify ownership
		user, err := GetUserFromContext(c)
		if err != nil {
			logrus.WithError(err).Error("Failed to get user from context")
			return fiber.NewError(fiber.StatusUnauthorized, "Authentication required")
		}

		if payment.UserID != user.ID && user.Role != "admin" {
			return fiber.NewError(fiber.StatusForbidden, "Access denied")
		}

		return c.JSON(payment)
	}
}

// HandleListPayments lists all payments for the current user
func HandleListPayments(repo *repository.PaymentRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get current user
		user, err := GetUserFromContext(c)
		if err != nil {
			logrus.WithError(err).Error("Failed to get user from context")
			return fiber.NewError(fiber.StatusUnauthorized, "Authentication required")
		}

		// Get pagination parameters
		page, err := strconv.ParseInt(c.Query("page", "1"), 10, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid page number")
		}
		limit, err := strconv.ParseInt(c.Query("limit", "10"), 10, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid limit value")
		}

		// Validate pagination parameters
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 10
		}

		// Get payments
		payments, total, err := repo.ListByUser(c.Context(), user.ID, page, limit)
		if err != nil {
			logrus.WithError(err).WithField("user_id", user.ID).Error("Failed to list payments")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve payment history")
		}

		return c.JSON(fiber.Map{
			"payments": payments,
			"total":    total,
			"page":     page,
			"limit":    limit,
		})
	}
}

// HandleStripeWebhook handles Stripe webhook events
func HandleStripeWebhook(repo *repository.PaymentRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Read request body
		payload, err := io.ReadAll(c.Request().BodyStream())
		if err != nil {
			logrus.WithError(err).Error("Failed to read webhook payload")
			return fiber.NewError(fiber.StatusBadRequest, "Failed to read request body")
		}

		// Verify webhook signature
		if config.AppConfig.StripeWebhook == "" {
			logrus.Error("Stripe webhook secret is not configured")
			return fiber.NewError(fiber.StatusInternalServerError, "Webhook configuration is missing")
		}

		event, err := webhook.ConstructEvent(payload, c.Get("Stripe-Signature"), config.AppConfig.StripeWebhook)
		if err != nil {
			logrus.WithError(err).Error("Invalid webhook signature")
			return fiber.NewError(fiber.StatusBadRequest, "Invalid webhook signature")
		}

		// Handle different event types
		switch event.Type {
		case "checkout.session.completed":
			var session stripe.CheckoutSession
			err := json.Unmarshal(event.Data.Raw, &session)
			if err != nil {
				logrus.WithError(err).Error("Failed to parse checkout session")
				return fiber.NewError(fiber.StatusBadRequest, "Failed to parse session data")
			}

			// Create payment record
			userID, err := primitive.ObjectIDFromHex(session.Customer.Metadata["user_id"])
			if err != nil {
				logrus.WithError(err).WithField("metadata", session.Customer.Metadata).Error("Invalid user ID in metadata")
				return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID in metadata")
			}

			payment := &models.Payment{
				UserID:        userID,
				Gateway:       "stripe",
				TransactionID: session.ID,
				Amount:        int(session.AmountTotal),
				Currency:      string(session.Currency),
				Status:        "completed",
				Timestamp:     time.Now(),
			}

			if err := repo.Create(c.Context(), payment); err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"user_id":        userID,
					"transaction_id": session.ID,
				}).Error("Failed to create payment record")
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to record payment")
			}

		case "customer.subscription.updated":
			var sub stripe.Subscription
			err := json.Unmarshal(event.Data.Raw, &sub)
			if err != nil {
				logrus.WithError(err).Error("Failed to parse subscription update")
				return fiber.NewError(fiber.StatusBadRequest, "Failed to parse subscription data")
			}

			// Update user's subscription status
			userID, err := primitive.ObjectIDFromHex(sub.Customer.Metadata["user_id"])
			if err != nil {
				logrus.WithError(err).WithField("metadata", sub.Customer.Metadata).Error("Invalid user ID in metadata")
				return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID in metadata")
			}

			subscription := models.Subscription{
				Status:           string(sub.Status),
				Plan:             string(sub.Items.Data[0].Price.Recurring.Interval),
				CurrentPeriodEnd: time.Unix(sub.CurrentPeriodEnd, 0),
			}

			if err := repo.UpdateSubscription(c.Context(), userID, subscription); err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"user_id": userID,
					"status":  sub.Status,
				}).Error("Failed to update subscription")
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update subscription")
			}

		case "customer.subscription.deleted":
			var sub stripe.Subscription
			err := json.Unmarshal(event.Data.Raw, &sub)
			if err != nil {
				logrus.WithError(err).Error("Failed to parse subscription deletion")
				return fiber.NewError(fiber.StatusBadRequest, "Failed to parse subscription data")
			}

			// Update user's subscription status
			userID, err := primitive.ObjectIDFromHex(sub.Customer.Metadata["user_id"])
			if err != nil {
				logrus.WithError(err).WithField("metadata", sub.Customer.Metadata).Error("Invalid user ID in metadata")
				return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID in metadata")
			}

			subscription := models.Subscription{
				Status:           "canceled",
				Plan:             string(sub.Items.Data[0].Price.Recurring.Interval),
				CurrentPeriodEnd: time.Unix(sub.CurrentPeriodEnd, 0),
			}

			if err := repo.UpdateSubscription(c.Context(), userID, subscription); err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"user_id": userID,
					"status":  "canceled",
				}).Error("Failed to update subscription")
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update subscription")
			}
		}

		return c.SendStatus(fiber.StatusOK)
	}
}

// HandleGetRegionalPricing gets pricing for a specific region
func HandleGetRegionalPricing(repo *repository.PaymentRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get region code from query params
		regionCode := c.Query("region")
		if regionCode == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Region code is required")
		}

		// Get pricing
		pricing, err := repo.GetRegionalPricing(c.Context(), regionCode)
		if err != nil {
			logrus.WithError(err).WithField("region", regionCode).Error("Failed to get regional pricing")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get pricing information")
		}
		if pricing == nil {
			return fiber.NewError(fiber.StatusNotFound, "Pricing not found for region")
		}

		return c.JSON(pricing)
	}
}
