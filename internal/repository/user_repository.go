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
	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		collection: database.Users,
	}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Insert user
	result, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return err
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByEmail finds a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByID finds a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"name":         user.Name,
			"email":        user.Email,
			"role":         user.Role,
			"is_verified":  user.IsVerified,
			"subscription": user.Subscription,
			"blocked":      user.Blocked,
			"updated_at":   user.UpdatedAt,
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": user.ID},
		update,
	)
	return err
}

// UpdateSubscription updates a user's subscription
func (r *UserRepository) UpdateSubscription(ctx context.Context, userID primitive.ObjectID, subscription models.Subscription) error {
	update := bson.M{
		"$set": bson.M{
			"subscription": subscription,
			"updated_at":   time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		update,
	)
	return err
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// VerifyPassword checks if the provided password matches the stored hash
func (r *UserRepository) VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// List returns a list of users with pagination
func (r *UserRepository) List(ctx context.Context, page, limit int64) ([]*models.User, int64, error) {
	skip := (page - 1) * limit

	// Get total count
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	// Find users with pagination
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []*models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// ListWithFilter returns a list of users with filtering and pagination
func (r *UserRepository) ListWithFilter(ctx context.Context, filter map[string]interface{}, page, limit int64) ([]*models.User, int64, error) {
	skip := (page - 1) * limit

	// Get total count with filter
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Find users with pagination and filter
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []*models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetUserStats returns user statistics
func (r *UserRepository) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total users
	totalUsers, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	stats["total_users"] = totalUsers

	// Verified users
	verifiedUsers, err := r.collection.CountDocuments(ctx, bson.M{"is_verified": true})
	if err != nil {
		return nil, err
	}
	stats["verified_users"] = verifiedUsers

	// Blocked users
	blockedUsers, err := r.collection.CountDocuments(ctx, bson.M{"blocked": true})
	if err != nil {
		return nil, err
	}
	stats["blocked_users"] = blockedUsers

	// Users by role
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id": "$role",
				"count": bson.M{
					"$sum": 1,
				},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var roleStats []bson.M
	if err = cursor.All(ctx, &roleStats); err != nil {
		return nil, err
	}

	roleCounts := make(map[string]int64)
	for _, stat := range roleStats {
		role := stat["_id"].(string)
		count := stat["count"].(int64)
		roleCounts[role] = count
	}
	stats["users_by_role"] = roleCounts

	// New users in last 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	newUsers, err := r.collection.CountDocuments(ctx, bson.M{
		"created_at": bson.M{
			"$gte": thirtyDaysAgo,
		},
	})
	if err != nil {
		return nil, err
	}
	stats["new_users_last_30_days"] = newUsers

	return stats, nil
}
