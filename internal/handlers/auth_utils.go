package handlers

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
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
	otpCode, err := generateOTP(6)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate OTP")
		return nil, err
	}

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

func generateOTP(length int) (string, error) {
	const digits = "0123456789"
	otp := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp[i] = digits[num.Int64()]
	}
	return string(otp), nil
}
