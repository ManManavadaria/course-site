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

type VideoRepository struct {
	collection *mongo.Collection
}

func NewVideoRepository() *VideoRepository {
	return &VideoRepository{
		collection: database.Videos,
	}
}

// Create creates a new video
func (r *VideoRepository) Create(ctx context.Context, video *models.Video) error {
	video.CreatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, video)
	if err != nil {
		return err
	}

	video.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID finds a video by ID
func (r *VideoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Video, error) {
	var video models.Video
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&video)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &video, nil
}

// ListByCourse returns a list of videos for a specific course
func (r *VideoRepository) ListByCourse(ctx context.Context, courseID primitive.ObjectID, page, limit int64) ([]*models.Video, int64, error) {
	skip := (page - 1) * limit

	// Get total count
	total, err := r.collection.CountDocuments(ctx, bson.M{"course_id": courseID})
	if err != nil {
		return nil, 0, err
	}

	// Find videos with pagination
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, bson.M{"course_id": courseID}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var videos []*models.Video
	if err = cursor.All(ctx, &videos); err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

// Update updates a video
func (r *VideoRepository) Update(ctx context.Context, video *models.Video) error {
	update := bson.M{
		"$set": bson.M{
			"title":       video.Title,
			"description": video.Description,
			"url":         video.URL,
			"thumbnail":   video.Thumbnail,
			"duration":    video.Duration,
			"is_paid":     video.IsPaid,
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": video.ID},
		update,
	)
	return err
}

// Delete deletes a video
func (r *VideoRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// UpdateWatchHistory updates or creates a watch history entry
func (r *VideoRepository) UpdateWatchHistory(ctx context.Context, history *models.WatchHistory) error {
	// Use upsert to create or update the watch history
	opts := options.Update().SetUpsert(true)
	update := bson.M{
		"$set": bson.M{
			"last_watched_at":  time.Now(),
			"progress_seconds": history.ProgressSeconds,
		},
	}

	_, err := database.WatchHistory.UpdateOne(
		ctx,
		bson.M{
			"user_id":  history.UserID,
			"video_id": history.VideoID,
		},
		update,
		opts,
	)
	return err
}

// GetWatchHistory gets the watch history for a user and video
func (r *VideoRepository) GetWatchHistory(ctx context.Context, userID, videoID primitive.ObjectID) (*models.WatchHistory, error) {
	var history models.WatchHistory
	err := database.WatchHistory.FindOne(ctx, bson.M{
		"user_id":  userID,
		"video_id": videoID,
	}).Decode(&history)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &history, nil
}

// ListWatchHistory gets all watch history entries for a user
func (r *VideoRepository) ListWatchHistory(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]*models.WatchHistory, int64, error) {
	skip := (page - 1) * limit

	// Get total count
	total, err := database.WatchHistory.CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, 0, err
	}

	// Find watch history with pagination
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.M{"last_watched_at": -1})

	cursor, err := database.WatchHistory.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var history []*models.WatchHistory
	if err = cursor.All(ctx, &history); err != nil {
		return nil, 0, err
	}

	return history, total, nil
}
