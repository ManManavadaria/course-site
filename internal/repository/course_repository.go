package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cource-api/internal/database"
	"cource-api/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CourseRepository struct {
	collection *mongo.Collection
	videoRepo  *VideoRepository
}

func NewCourseRepository(videoRepo *VideoRepository) *CourseRepository {
	return &CourseRepository{
		collection: database.Courses,
		videoRepo:  videoRepo,
	}
}

// Create creates a new course
func (r *CourseRepository) Create(ctx context.Context, course *models.Course) error {
	course.CreatedAt = time.Now()
	course.UpdatedAt = time.Now()
	course.VideoOrder = []primitive.ObjectID{} // Initialize empty video order

	result, err := r.collection.InsertOne(ctx, course)
	if err != nil {
		return err
	}

	course.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID finds a course by ID
func (r *CourseRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Course, error) {
	var course models.Course
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&course)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &course, nil
}

// List returns a list of courses with pagination
func (r *CourseRepository) List(ctx context.Context, page, limit int64) ([]*models.Course, int64, error) {
	skip := (page - 1) * limit

	// Get total count
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	// Find courses with pagination
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var courses []*models.Course
	if err = cursor.All(ctx, &courses); err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

// Update updates a course
func (r *CourseRepository) Update(ctx context.Context, course *models.Course) error {
	course.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"title":         course.Title,
			"subtitle":      course.SubTitle,
			"description":   course.Description,
			"thumbnail_url": course.ThumbnailURL,
			"video_order":   course.VideoOrder,
			"is_paid":       course.IsPaid,
			"skills":        course.Skills,
			"author":        course.Author,
			"updated_at":    course.UpdatedAt,
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": course.ID},
		update,
	)
	return err
}

// Delete deletes a course
func (r *CourseRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// AddVideoToCourse adds a video to a course at a specific position
func (r *CourseRepository) AddVideoToCourse(ctx context.Context, courseID primitive.ObjectID, videoID primitive.ObjectID, position int) error {
	// Get the course first
	course, err := r.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course == nil {
		return errors.New("course not found")
	}

	// Get current video order
	currentOrder := course.VideoOrder

	// Validate position
	if position < 0 || position > len(currentOrder) {
		return errors.New("invalid position")
	}

	// Create new order array with the video inserted at the specified position
	newOrder := make([]primitive.ObjectID, len(currentOrder)+1)
	copy(newOrder[:position], currentOrder[:position])
	newOrder[position] = videoID
	copy(newOrder[position+1:], currentOrder[position:])

	// Update the course's video order
	update := bson.M{
		"$set": bson.M{
			"video_order": newOrder,
			"updated_at":  time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": courseID},
		update,
	)
	return err
}

// ReorderVideos reorders videos within a course
func (r *CourseRepository) ReorderVideos(ctx context.Context, courseID primitive.ObjectID, newOrder []primitive.ObjectID) error {
	// Get the course first
	course, err := r.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course == nil {
		return errors.New("course not found")
	}

	// Validate that all videos in new order exist in the course
	currentOrder := course.VideoOrder
	videoMap := make(map[string]bool)
	for _, v := range currentOrder {
		videoMap[v.Hex()] = true
	}

	for _, v := range newOrder {
		if !videoMap[v.Hex()] {
			return errors.New("invalid video ID in new order")
		}
	}

	// Update the course's video order
	update := bson.M{
		"$set": bson.M{
			"video_order": newOrder,
			"updated_at":  time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": courseID},
		update,
	)
	return err
}

// RemoveVideoFromCourse removes a video from a course
func (r *CourseRepository) RemoveVideoFromCourse(ctx context.Context, courseID primitive.ObjectID, videoID primitive.ObjectID) error {
	// Get the course first
	course, err := r.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course == nil {
		return errors.New("course not found")
	}

	// Create new order array without the specified video
	currentOrder := course.VideoOrder
	if len(currentOrder) == 0 {
		return fmt.Errorf("Course not having any videos")
	}
	newOrder := make([]primitive.ObjectID, 0, len(currentOrder)-1)

	if len(currentOrder) == 1 && currentOrder[0] == videoID {
		newOrder = nil
	} else {
		for _, v := range currentOrder {
			if v != videoID {
				newOrder = append(newOrder, v)
			}
		}
	}

	// Update the course's video order
	update := bson.M{
		"$set": bson.M{
			"video_order": newOrder,
			"updated_at":  time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": courseID},
		update,
	)
	return err
}

// GetVideosInOrder returns videos in the correct order for a course
func (r *CourseRepository) GetVideosInOrder(ctx context.Context, courseID primitive.ObjectID) ([]*models.Video, error) {
	// Get the course first
	course, err := r.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, errors.New("course not found")
	}

	// Get video order
	videoOrder := course.VideoOrder
	if len(videoOrder) == 0 {
		return []*models.Video{}, nil
	}

	// Get all videos in the course
	videos := make([]*models.Video, len(videoOrder))
	for i, videoID := range videoOrder {
		video, err := r.videoRepo.GetByID(ctx, videoID)
		if err != nil {
			return nil, err
		}
		if video == nil {
			return nil, fmt.Errorf("video not found: %s", videoID.Hex())
		}
		videos[i] = video
	}

	return videos, nil
}
