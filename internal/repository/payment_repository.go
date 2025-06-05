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

type PaymentRepository struct {
	collection *mongo.Collection
}

func NewPaymentRepository() *PaymentRepository {
	return &PaymentRepository{
		collection: database.Payments,
	}
}

// Create creates a new payment record
func (r *PaymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	payment.Timestamp = time.Now()

	result, err := r.collection.InsertOne(ctx, payment)
	if err != nil {
		return err
	}

	payment.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID finds a payment by ID
func (r *PaymentRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Payment, error) {
	var payment models.Payment
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&payment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &payment, nil
}

// GetByTransactionID finds a payment by transaction ID
func (r *PaymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error) {
	var payment models.Payment
	err := r.collection.FindOne(ctx, bson.M{"transaction_id": transactionID}).Decode(&payment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &payment, nil
}

// ListByUser returns a list of payments for a specific user
func (r *PaymentRepository) ListByUser(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]*models.Payment, int64, error) {
	skip := (page - 1) * limit

	// Get total count
	total, err := r.collection.CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, 0, err
	}

	// Find payments with pagination
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.M{"timestamp": -1})

	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var payments []*models.Payment
	if err = cursor.All(ctx, &payments); err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

// UpdateStatus updates a payment's status
func (r *PaymentRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status string) error {
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		update,
	)
	return err
}

// GetRegionalPricing gets pricing for a specific region
func (r *PaymentRepository) GetRegionalPricing(ctx context.Context, regionCode string) (*models.RegionalPricing, error) {
	var pricing models.RegionalPricing
	err := database.RegionalPricing.FindOne(ctx, bson.M{"region_code": regionCode}).Decode(&pricing)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &pricing, nil
}

// UpdateRegionalPricing updates pricing for a specific region
func (r *PaymentRepository) UpdateRegionalPricing(ctx context.Context, pricing *models.RegionalPricing) error {
	opts := options.Update().SetUpsert(true)
	update := bson.M{
		"$set": bson.M{
			"currency":        pricing.Currency,
			"monthly_price":   pricing.MonthlyPrice,
			"yearly_price":    pricing.YearlyPrice,
			"currency_symbol": pricing.CurrencySymbol,
		},
	}

	_, err := database.RegionalPricing.UpdateOne(
		ctx,
		bson.M{"region_code": pricing.RegionCode},
		update,
		opts,
	)
	return err
}

// ListRegionalPricing returns a list of all regional pricing
func (r *PaymentRepository) ListRegionalPricing(ctx context.Context) ([]*models.RegionalPricing, error) {
	cursor, err := database.RegionalPricing.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var pricing []*models.RegionalPricing
	if err = cursor.All(ctx, &pricing); err != nil {
		return nil, err
	}

	return pricing, nil
}

// UpdateSubscription updates a user's subscription
func (r *PaymentRepository) UpdateSubscription(ctx context.Context, userID primitive.ObjectID, subscription models.Subscription) error {
	update := bson.M{
		"$set": bson.M{
			"subscription": subscription,
			"updated_at":   time.Now(),
		},
	}

	_, err := database.Users.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		update,
	)
	return err
}
