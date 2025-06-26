package handlers

import (
	"cource-api/internal/aws"
	"cource-api/internal/repository"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// HandleGeneratePresignedURL generates a pre-signed URL for video/thumbnail upload
func HandleVideoGeneratePresignedURL() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get current user
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		// Parse request body
		var req struct {
			FileName    string `json:"file_name"`
			FileType    string `json:"file_type"`
			ContentType string `json:"content_type"`
		}

		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate request
		if req.FileName == "" {
			return fiber.NewError(fiber.StatusBadRequest, "File name is required")
		}
		if req.FileType == "" {
			return fiber.NewError(fiber.StatusBadRequest, "File type is required")
		}
		if req.ContentType == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Content type is required")
		}

		fmt.Printf("%+v\n", user)

		// Generate a unique file key
		fileKey := fmt.Sprintf("%s/%s/%s", req.FileType, user.ID.Hex(), req.FileName)

		// Generate pre-signed URL
		presignedURL, err := aws.S3C.GeneratePresignedURL(fileKey, req.ContentType, 1)
		if err != nil {
			logrus.WithError(err).Error("Failed to generate pre-signed URL")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate upload URL")
		}

		return c.JSON(fiber.Map{
			"upload_url": presignedURL,
			"file_key":   fileKey,
		})
	}
}

// HandleGeneratePresignedURL generates a pre-signed URL for video/thumbnail upload
func HandleThumbnailGeneratePresignedURL() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get current user
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		// Parse request body
		var req struct {
			FileName    string `json:"file_name"`
			FileType    string `json:"file_type"`
			ContentType string `json:"content_type"`
		}

		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate request
		if req.FileName == "" {
			return fiber.NewError(fiber.StatusBadRequest, "File name is required")
		}
		if req.FileType == "" {
			return fiber.NewError(fiber.StatusBadRequest, "File type is required")
		}
		if req.ContentType == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Content type is required")
		}

		// Generate a unique file key
		fileKey := fmt.Sprintf("%s/%s/%s", req.FileType, user.ID.Hex(), req.FileName)

		// Generate pre-signed URL for upload
		presignedURL, err := aws.S3C.GenerateThumbnailUploadURL(fileKey, req.ContentType, 1)
		if err != nil {
			logrus.WithError(err).Error("Failed to generate pre-signed URL")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate upload URL")
		}

		// Generate the public URL for the thumbnail
		publicURL := aws.S3C.GetThumbnailURL(fileKey)

		return c.JSON(fiber.Map{
			"upload_url": presignedURL,
			"file_key":   fileKey,
			"public_url": publicURL,
		})
	}
}

// HandleUploadComplete handles the notification of upload completion
func HandleUploadComplete(repo *repository.VideoRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get current user
		_, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		// Parse request body
		var req struct {
			FileKey string `json:"file_key"`
			Type    string `json:"type"` // "video" or "thumbnail"
		}

		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate request
		if req.FileKey == "" {
			return fiber.NewError(fiber.StatusBadRequest, "File key is required")
		}
		if req.Type == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Type is required")
		}

		// Create S3 client
		s3Client, err := aws.NewS3Client()
		if err != nil {
			logrus.WithError(err).Error("Failed to create S3 client")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify upload")
		}

		// Verify file exists in S3
		exists, err := s3Client.FileExists(req.FileKey)
		if err != nil {
			logrus.WithError(err).Error("Failed to verify file existence")
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify upload")
		}
		if !exists {
			return fiber.NewError(fiber.StatusBadRequest, "File not found in S3")
		}

		// Generate the public URL for the file
		fileURL := s3Client.GetPublicURL(req.FileKey)

		return c.JSON(fiber.Map{
			"file_url": fileURL,
			"type":     req.Type,
		})
	}
}
