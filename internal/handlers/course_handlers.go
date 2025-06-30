package handlers

import (
	"cource-api/internal/aws"
	"cource-api/internal/models"
	"cource-api/internal/repository"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HandleListCourses lists all courses with pagination
func HandleListCourses(repo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get pagination parameters
		page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
		limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)

		// Get courses
		courses, total, err := repo.List(c.Context(), page, limit, true)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to list courses")
		}

		return c.JSON(fiber.Map{
			"courses": courses,
			"total":   total,
			"page":    page,
			"limit":   limit,
		})
	}
}

// HandleListCourses lists all courses with pagination
func HandleAdminListCourses(repo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get pagination parameters
		page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
		limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)

		// Get courses
		courses, total, err := repo.List(c.Context(), page, limit, false)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to list courses")
		}

		return c.JSON(fiber.Map{
			"courses": courses,
			"total":   total,
			"page":    page,
			"limit":   limit,
		})
	}
}

// HandleCreateCourse creates a new course
func HandleCreateCourse(repo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get current user
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		// Parse request body
		var req struct {
			Title        string   `json:"title"`
			SubTitle     string   `json:"subtitle"`
			Description  string   `json:"description"`
			IsPaid       bool     `json:"is_paid"`
			Skills       []string `json:"skills"`
			Author       string   `json:"author"`
			ThumbnailURL string   `json:"thumbnail_url"`
			IsPublic     bool     `json:"is_public"`
		}

		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		//NOTE: handle thumbnail upload logic and add the thumbnail url to the course document

		// Create course
		course := &models.Course{
			Title:        req.Title,
			SubTitle:     req.SubTitle,
			Description:  req.Description,
			IsPaid:       req.IsPaid,
			IsPublic:     req.IsPublic,
			Skills:       req.Skills,
			Author:       req.Author,
			ThumbnailURL: req.ThumbnailURL,
			CreatedBy:    user.ID,
			VideoOrder:   []primitive.ObjectID{},
		}

		if err := repo.Create(c.Context(), course); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create course")
		}

		return c.JSON(course)
	}
}

// HandleGetCourse gets a course by ID
func HandleGetCourse(repo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get course ID from params
		courseID := c.Params("id")
		if courseID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Course ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(courseID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid course ID format")
		}

		// Get course
		course, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get course")
		}
		if course == nil {
			return fiber.NewError(fiber.StatusNotFound, "Course not found")
		}

		// Get videos in order
		videos, err := repo.GetVideosInOrder(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get course videos")
		}

		// Add videos to response
		response := fiber.Map{
			"course": course,
			"videos": videos,
		}

		return c.JSON(response)
	}
}

// HandleUpdateCourse updates a course
func HandleUpdateCourse(repo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get course ID from params
		courseID := c.Params("id")
		if courseID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Course ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(courseID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid course ID format")
		}

		// Get course
		course, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get course")
		}
		if course == nil {
			return fiber.NewError(fiber.StatusNotFound, "Course not found")
		}

		// Parse request body
		var updateData struct {
			Title        string   `json:"title"`
			SubTitle     string   `json:"subtitle"`
			Description  string   `json:"description"`
			IsPaid       bool     `json:"is_paid"`
			IsPublic     bool     `json:"is_public"`
			Skills       []string `json:"skills"`
			Author       string   `json:"author"`
			ThumbnailURL string   `json:"thumbnail_url"`
		}

		if err := c.BodyParser(&updateData); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		//NOTE: handle the s3 thumbnail update logic and update the url in the course document

		// Update course fields
		if updateData.Title != "" {
			course.Title = updateData.Title
		}
		course.SubTitle = updateData.SubTitle
		course.Description = updateData.Description
		if updateData.ThumbnailURL != course.ThumbnailURL {
			if err := aws.S3C.DeleteFile(course.ThumbnailURL); err != nil {
				logrus.Error(err)
			}
			course.ThumbnailURL = updateData.ThumbnailURL
		}
		course.IsPaid = updateData.IsPaid
		course.Skills = nil
		course.Skills = updateData.Skills
		course.Author = updateData.Author
		course.IsPublic = updateData.IsPublic

		// Update course
		if err := repo.Update(c.Context(), course); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update course")
		}

		return c.JSON(course)
	}
}

// HandleDeleteCourse deletes a course
func HandleDeleteCourse(repo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get course ID from params
		courseID := c.Params("id")
		if courseID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Course ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(courseID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid course ID format")
		}

		//NOTE: Remove the couse reference from the corresponding videos as well

		// Delete course
		if err := repo.Delete(c.Context(), objectID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete course")
		}

		return c.SendStatus(fiber.StatusOK)
	}
}

// HandleReorderVideos reorders videos in a course
func HandleReorderVideos(repo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get course ID from params
		courseID := c.Params("id")
		if courseID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Course ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(courseID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid course ID format")
		}

		// Parse request body
		var req struct {
			VideoOrder []string `json:"video_order"`
		}

		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Convert video IDs to ObjectIDs
		videoOrder := make([]primitive.ObjectID, len(req.VideoOrder))
		for i, id := range req.VideoOrder {
			videoID, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid video ID format")
			}
			videoOrder[i] = videoID
		}

		if err := repo.ReorderVideos(c.Context(), objectID, videoOrder); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to reorder videos")
		}

		return c.SendStatus(fiber.StatusOK)
	}
}

// HandleAddVideoToCourse adds a video to a course
func HandleAddVideoToCourse(repo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get course ID from params
		courseID := c.Params("id")
		if courseID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Course ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(courseID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid course ID format")
		}

		// Parse request body
		var req struct {
			VideoID  string `json:"video_id"`
			Position int    `json:"position"`
		}

		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Convert video ID to ObjectID
		videoID, err := primitive.ObjectIDFromHex(req.VideoID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid video ID format")
		}

		// Add video to course
		if err := repo.AddVideoToCourse(c.Context(), objectID, videoID, req.Position); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to add video to course")
		}

		return c.SendStatus(fiber.StatusOK)
	}
}

// HandleRemoveVideoFromCourse removes a video from a course
func HandleRemoveVideoFromCourse(repo *repository.CourseRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get course ID from params
		courseID := c.Params("id")
		if courseID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Course ID is required")
		}

		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(courseID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid course ID format")
		}

		// Get video ID from params
		videoID := c.Params("video_id")
		if videoID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Video ID is required")
		}

		// Convert video ID to ObjectID
		videoObjectID, err := primitive.ObjectIDFromHex(videoID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid video ID format")
		}

		// Remove video from course
		if err := repo.RemoveVideoFromCourse(c.Context(), objectID, videoObjectID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to remove video from course")
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}
