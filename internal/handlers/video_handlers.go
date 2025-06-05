package handlers

import (
	"cource-api/internal/models"
	"cource-api/internal/repository"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
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
func HandleCreateVideo(repo *repository.VideoRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse request body
		var video models.Video
		if err := c.BodyParser(&video); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate required fields
		if video.Title == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Title is required")
		}
		if video.CourseID.IsZero() {
			return fiber.NewError(fiber.StatusBadRequest, "Course ID is required")
		}

		// Set creation time
		video.CreatedAt = time.Now()

		// Create video
		if err := repo.Create(c.Context(), &video); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create video")
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

		return c.JSON(video)
	}
}

// HandleUpdateVideo updates a video
func HandleUpdateVideo(repo *repository.VideoRepository) fiber.Handler {
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
			Title       string `json:"title"`
			Description string `json:"description"`
			URL         string `json:"url"`
			Thumbnail   string `json:"thumbnail"`
			Duration    int    `json:"duration"`
			IsPaid      bool   `json:"is_paid"`
		}

		if err := c.BodyParser(&updateData); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Update video fields
		if updateData.Title != "" {
			video.Title = updateData.Title
		}
		if updateData.Description != "" {
			video.Description = updateData.Description
		}
		if updateData.URL != "" {
			video.URL = updateData.URL
		}
		if updateData.Thumbnail != "" {
			video.Thumbnail = updateData.Thumbnail
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
func HandleDeleteVideo(repo *repository.VideoRepository) fiber.Handler {
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

		// Delete video
		if err := repo.Delete(c.Context(), objectID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete video")
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
