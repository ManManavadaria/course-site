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

type OTPRepository struct {
	collection *mongo.Collection
}

func NewOTPRepository() *OTPRepository {
	return &OTPRepository{
		collection: database.OTPs,
	}
}

// Create creates a new OTP
func (r *OTPRepository) Create(ctx context.Context, otp *models.OTP) error {
	otp.CreatedAt = time.Now()
	otp.ExpiresAt = time.Now().Add(15 * time.Minute) // OTP expires in 15 minutes

	result, err := r.collection.InsertOne(ctx, otp)
	if err != nil {
		return err
	}

	otp.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetLatestOTP gets the latest unused OTP for an email
func (r *OTPRepository) GetLatestOTP(ctx context.Context, email, otpType string) (*models.OTP, error) {
	var otp models.OTP
	err := r.collection.FindOne(ctx, bson.M{
		"email": email,
		"type":  otpType,
		"used":  false,
		"expires_at": bson.M{
			"$gt": time.Now(),
		},
	}, options.FindOne().SetSort(bson.M{"created_at": -1})).Decode(&otp)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &otp, nil
}

// MarkAsUsed marks an OTP as used
func (r *OTPRepository) MarkAsUsed(ctx context.Context, id primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"used": true,
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		update,
	)
	return err
}

// DeleteExpiredOTPs deletes expired OTPs
func (r *OTPRepository) DeleteExpiredOTPs(ctx context.Context) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{
		"expires_at": bson.M{
			"$lt": time.Now(),
		},
	})
	return err
}
