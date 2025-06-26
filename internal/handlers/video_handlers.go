package handlers

import (
	"cource-api/internal/aws"
	"cource-api/internal/models"
	"cource-api/internal/repository"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HandleListVideos lists all videos with pagination
func HandleListVideos(repo *repository.VideoRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get pagination parameters
		page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
		limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)

		// Get course ID from query params if provided
		courseID := c.Query("course_id")
		var videos []*models.Video
		var total int64
		var err error

		if courseID != "" {
			// Convert course ID to ObjectID
			objectID, err := primitive.ObjectIDFromHex(courseID)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid course ID format")
			}
			videos, total, err = repo.ListByCourse(c.Context(), objectID, page, limit)
		}

		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to list videos")
		}

		return c.JSON(fiber.Map{
			"videos": videos,
			"total":  total,
			"page":   page,
			"limit":  limit,
		})
	}
}

// HandleCreateVideo creates a new video
func HandleCreateVideo(repo *repository.VideoRepository, courseRepo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse request body
		var req struct {
			Title        string             `json:"title"`
			Description  string             `json:"description"`
			VideoURL     string             `json:"video_url"`     // Direct S3 URL for video
			ThumbnailURL string             `json:"thumbnail_url"` // Direct S3 URL for thumbnail
			Duration     int                `json:"duration"`
			IsPaid       bool               `json:"is_paid"`
			CourseID     primitive.ObjectID `json:"course_id"`
		}

		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate required fields
		if req.Title == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Title is required")
		}
		if req.VideoURL == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Video URL is required")
		}
		if req.ThumbnailURL == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Thumbnail URL is required")
		}
		if req.CourseID.IsZero() {
			return fiber.NewError(fiber.StatusBadRequest, "Course ID is required")
		}

		// Check if course exists
		course, err := courseRepo.GetByID(c.Context(), req.CourseID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify course")
		}
		if course == nil {
			return fiber.NewError(fiber.StatusNotFound, "Course not found")
		}

		// Create video object
		video := &models.Video{
			Title:       req.Title,
			Description: req.Description,
			URL:         req.VideoURL,
			Thumbnail:   req.ThumbnailURL,
			Duration:    req.Duration,
			IsPaid:      req.IsPaid,
			CourseID:    req.CourseID,
			CreatedAt:   time.Now(),
		}

		// Create video
		if err := repo.Create(c.Context(), video); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create video")
		}

		// Add video to course's video order
		if err := courseRepo.AddVideoToCourse(c.Context(), video.CourseID, video.ID, len(course.VideoOrder)); err != nil {
			// If adding to course fails, delete the video
			_ = repo.Delete(c.Context(), video.ID)
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to add video to course")
		}

		return c.Status(fiber.StatusCreated).JSON(video)
	}
}

// HandleGetVideo gets a specific video by ID
func HandleGetVideo(repo *repository.VideoRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get video ID from params
		videoID := c.Params("id")
		if videoID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Video ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(videoID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid video ID format")
		}

		// Get video
		video, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get video")
		}
		if video == nil {
			return fiber.NewError(fiber.StatusNotFound, "Video not found")
		}

		presignedURL, err := aws.S3C.GenerateWatchURL(video.URL, 12)
		if err != nil {
			logrus.WithError(err).Error("Failed to generate pre-signed URL")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate upload URL")
		}

		video.URL = presignedURL

		return c.JSON(video)
	}
}

// HandleUpdateVideo updates a video
func HandleUpdateVideo(repo *repository.VideoRepository, courseRepo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get video ID from params
		videoID := c.Params("id")
		if videoID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Video ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(videoID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid video ID format")
		}

		// Get existing video
		video, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get video")
		}
		if video == nil {
			return fiber.NewError(fiber.StatusNotFound, "Video not found")
		}

		// Parse update data
		var updateData struct {
			Title        string             `json:"title"`
			Description  string             `json:"description"`
			VideoURL     string             `json:"video_url"`     // Direct S3 URL for video
			ThumbnailURL string             `json:"thumbnail_url"` // Direct S3 URL for thumbnail
			Duration     int                `json:"duration"`
			IsPaid       bool               `json:"is_paid"`
			CourseID     primitive.ObjectID `json:"course_id"`
		}

		if err := c.BodyParser(&updateData); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Handle course change if needed
		if video.CourseID != updateData.CourseID {
			course, err := courseRepo.GetByID(c.Context(), updateData.CourseID)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify course")
			}
			if course == nil {
				return fiber.NewError(fiber.StatusNotFound, "Course not found")
			}

			//NOTE: Solve the issue of remove and add video to new course
			if err := courseRepo.RemoveVideoFromCourse(c.Context(), video.CourseID, video.ID); err != nil {
				logrus.Error(err)
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to remove video from old course")
			}

			// Add video to new course
			if err := courseRepo.AddVideoToCourse(c.Context(), updateData.CourseID, video.ID, len(course.VideoOrder)); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to add video to new course")
			}

			video.CourseID = updateData.CourseID
		}

		// Update video fields
		if updateData.Title != "" {
			video.Title = updateData.Title
		}
		if updateData.Description != "" {
			video.Description = updateData.Description
		}
		if updateData.VideoURL != "" {
			video.URL = updateData.VideoURL
		}
		if updateData.ThumbnailURL != "" {
			video.Thumbnail = updateData.ThumbnailURL
		}
		if updateData.Duration > 0 {
			video.Duration = updateData.Duration
		}
		video.IsPaid = updateData.IsPaid

		// Update video
		if err := repo.Update(c.Context(), video); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update video")
		}

		return c.JSON(video)
	}
}

// HandleDeleteVideo deletes a video
func HandleDeleteVideo(repo *repository.VideoRepository, courseRepo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get video ID from params
		videoID := c.Params("id")
		if videoID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Video ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(videoID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid video ID format")
		}

		// Get existing video
		video, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get video")
		}
		if video == nil {
			return fiber.NewError(fiber.StatusNotFound, "Video not found")
		}

		// Delete video file from S3
		if err := aws.S3C.DeleteFile(video.URL); err != nil {
			logrus.WithError(err).WithField("video_id", videoID).Error("Failed to delete video file from S3")
			// Continue with deletion even if S3 deletion fails
		}

		// Delete thumbnail from S3
		if err := aws.S3C.DeleteThumbnail(video.Thumbnail); err != nil {
			logrus.WithError(err).WithField("video_id", videoID).Error("Failed to delete thumbnail from S3")
			// Continue with deletion even if S3 deletion fails
		}

		// Delete video from database
		if err := repo.Delete(c.Context(), objectID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete video")
		}

		// Remove video from course's video order
		if err := courseRepo.RemoveVideoFromCourse(c.Context(), video.CourseID, video.ID); err != nil {
			logrus.WithError(err).WithField("video_id", videoID).Error("Failed to remove video from course")
			// Continue even if removing from course fails
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// HandleUpdateWatchHistory updates or creates a watch history entry
func HandleUpdateWatchHistory(repo *repository.VideoRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get current user
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		// Get video ID from params
		videoID := c.Params("id")
		if videoID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Video ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(videoID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid video ID format")
		}

		// Parse request body
		var updateData struct {
			ProgressSeconds int `json:"progress_seconds"`
		}

		if err := c.BodyParser(&updateData); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Create watch history entry
		history := &models.WatchHistory{
			UserID:          user.ID,
			VideoID:         objectID,
			LastWatchedAt:   time.Now(),
			ProgressSeconds: updateData.ProgressSeconds,
		}

		// Update watch history
		if err := repo.UpdateWatchHistory(c.Context(), history); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update watch history")
		}

		return c.JSON(history)
	}
}

// HandleGetWatchHistory gets the watch history for a user
func HandleGetWatchHistory(repo *repository.VideoRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get current user
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		// Get pagination parameters
		page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
		limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)

		// Get watch history
		history, total, err := repo.ListWatchHistory(c.Context(), user.ID, page, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get watch history")
		}

		return c.JSON(fiber.Map{
			"history": history,
			"total":   total,
			"page":    page,
			"limit":   limit,
		})
	}
}
