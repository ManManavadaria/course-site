package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"cource-api/internal/models"
	"cource-api/internal/repository"

	"github.com/sirupsen/logrus"
)

var (
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")
)

// GenerateAndSaveOTP generates a new OTP and saves it to the database
func GenerateAndSaveOTP(ctx context.Context, otpRepo *repository.OTPRepository, email string, otpType string) (*models.OTP, error) {
	// Generate OTP
	otpBytes := make([]byte, 3)
	if _, err := rand.Read(otpBytes); err != nil {
		logrus.WithError(err).Error("Failed to generate OTP")
		return nil, err
	}
	otpCode := hex.EncodeToString(otpBytes)[:6]

	// Create OTP record
	otp := &models.OTP{
		Email:     email,
		Code:      otpCode,
		Type:      otpType,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(15 * time.Minute), // OTP expires in 15 minutes
		Used:      false,
	}

	if err := otpRepo.Create(ctx, otp); err != nil {
		logrus.WithError(err).Error("Failed to save OTP")
		return nil, err
	}

	// TODO: Send OTP via email
	logrus.WithFields(logrus.Fields{
		"email": email,
		"otp":   otpCode,
	}).Info("OTP generated and saved")

	return otp, nil
}
