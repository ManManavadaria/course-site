package repository

import (
	"context"
	"errors"
	"time"

	"cource-api/internal/database"
	"cource-api/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SubscriptionRepository struct {
	collection *mongo.Collection
}

func NewSubscriptionRepository() *SubscriptionRepository {
	return &SubscriptionRepository{
		collection: database.Subscriptions,
	}
}

// Create creates a new subscription
func (r *SubscriptionRepository) Create(ctx context.Context, subscription *models.Subscription) error {
	subscription.CreatedAt = time.Now()
	subscription.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, subscription)
	if err != nil {
		return err
	}

	subscription.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID finds a subscription by ID
func (r *SubscriptionRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&subscription)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &subscription, nil
}

// ListByUser returns a list of subscriptions for a specific user
func (r *SubscriptionRepository) ListByUser(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]*models.Subscription, int64, error) {
	skip := (page - 1) * limit

	// Get total count
	total, err := r.collection.CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, 0, err
	}

	// Find subscriptions with pagination
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var subscriptions []*models.Subscription
	if err = cursor.All(ctx, &subscriptions); err != nil {
		return nil, 0, err
	}

	return subscriptions, total, nil
}

// Update updates a subscription
func (r *SubscriptionRepository) Update(ctx context.Context, subscription *models.Subscription) error {
	subscription.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"status":               subscription.Status,
			"plan":                 subscription.Plan,
			"region":               subscription.Region,
			"currency":             subscription.Currency,
			"amount":               subscription.Amount,
			"current_period_start": subscription.CurrentPeriodStart,
			"current_period_end":   subscription.CurrentPeriodEnd,
			"cancel_at_period_end": subscription.CancelAtPeriodEnd,
			"canceled_at":          subscription.CanceledAt,
			"trial_start":          subscription.TrialStart,
			"trial_end":            subscription.TrialEnd,
			"payment_method_id":    subscription.PaymentMethodID,
			"customer_id":          subscription.CustomerID,
			"subscription_id":      subscription.SubscriptionID,
			"last_payment_status":  subscription.LastPaymentStatus,
			"last_payment_date":    subscription.LastPaymentDate,
			"next_billing_date":    subscription.NextBillingDate,
			"auto_renew":           subscription.AutoRenew,
			"updated_at":           subscription.UpdatedAt,
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": subscription.ID},
		update,
	)
	return err
}

// Delete deletes a subscription
func (r *SubscriptionRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// GetActiveSubscription gets the active subscription for a user
func (r *SubscriptionRepository) GetActiveSubscription(ctx context.Context, userID primitive.ObjectID) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.collection.FindOne(ctx, bson.M{
		"user_id": userID,
		"status": bson.M{
			"$in": []string{"active", "trial"},
		},
		"current_period_end": bson.M{
			"$gt": time.Now(),
		},
	}).Decode(&subscription)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &subscription, nil
}

// UpdatePaymentInfo updates payment-related information for a subscription
func (r *SubscriptionRepository) UpdatePaymentInfo(ctx context.Context, subscriptionID primitive.ObjectID, paymentInfo map[string]interface{}) error {
	update := bson.M{
		"$set": bson.M{
			"payment_method_id":   paymentInfo["payment_method_id"],
			"customer_id":         paymentInfo["customer_id"],
			"subscription_id":     paymentInfo["subscription_id"],
			"last_payment_status": paymentInfo["last_payment_status"],
			"last_payment_date":   paymentInfo["last_payment_date"],
			"next_billing_date":   paymentInfo["next_billing_date"],
			"updated_at":          time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": subscriptionID},
		update,
	)
	return err
}
