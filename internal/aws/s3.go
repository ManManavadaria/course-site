package aws

import (
	"context"
	"cource-api/internal/config"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client          *s3.Client
	bucketName      string
	thumbnailBucket string
	region          string
}

var S3C *S3Client

// NewS3Client creates a new S3 client
func NewS3Client() (*S3Client, error) {
	// Create custom credentials
	customCredentials := credentials.NewStaticCredentialsProvider(
		config.AppConfig.AWSAccessKeyID,
		config.AppConfig.AWSSecretAccessKey,
		"",
	)

	// Load AWS configuration with custom credentials
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(config.AppConfig.AWSRegion),
		awsconfig.WithCredentialsProvider(customCredentials),
	)
	if err != nil {
		return nil, err
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	log.Println("Connected to AWS s3!")

	return &S3Client{
		client:          client,
		bucketName:      config.AppConfig.AWSBucketName,
		thumbnailBucket: config.AppConfig.AWSThumbnailBucket,
		region:          config.AppConfig.AWSRegion,
	}, nil
}

// GeneratePresignedURL generates a pre-signed URL for uploading a file
func (s *S3Client) GeneratePresignedURL(fileKey, contentType string, hours float64) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	expirationDuration := time.Hour * time.Duration(hours)
	fmt.Println(expirationDuration)

	presignedURL, err := presignClient.PresignPutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(fileKey),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expirationDuration))

	if err != nil {
		return "", err
	}

	return presignedURL.URL, nil
}

// GenerateThumbnailUploadURL generates a pre-signed URL for uploading a thumbnail
func (s *S3Client) GenerateThumbnailUploadURL(fileKey, contentType string, hours float64) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	expirationDuration := time.Hour * time.Duration(hours)

	presignedURL, err := presignClient.PresignPutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(s.thumbnailBucket),
		Key:         aws.String(fileKey),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expirationDuration))

	if err != nil {
		return "", err
	}

	return presignedURL.URL, nil
}

// GenerateWatchURL generates a pre-signed URL for watching a video
func (s *S3Client) GenerateWatchURL(fileKey string, hours float64) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	expirationDuration := time.Hour * time.Duration(hours)

	presignedURL, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fileKey),
	}, s3.WithPresignExpires(expirationDuration))

	if err != nil {
		return "", err
	}

	return presignedURL.URL, nil
}

// FileExists checks if a file exists in S3
func (s *S3Client) FileExists(fileKey string) (bool, error) {
	_, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fileKey),
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

// ThumbnailExists checks if a thumbnail exists in the thumbnail bucket
func (s *S3Client) ThumbnailExists(fileKey string) (bool, error) {
	_, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(s.thumbnailBucket),
		Key:    aws.String(fileKey),
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

// DeleteFile deletes a file from the main S3 bucket
func (s *S3Client) DeleteFile(fileKey string) error {
	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fileKey),
	})
	return err
}

// DeleteThumbnail deletes a file from the thumbnail bucket
func (s *S3Client) DeleteThumbnail(fileKey string) error {
	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.thumbnailBucket),
		Key:    aws.String(fileKey),
	})
	return err
}

// GetPublicURL generates the public URL for a file
func (s *S3Client) GetPublicURL(fileKey string) string {
	return "https://" + s.bucketName + ".s3." + s.region + ".amazonaws.com/" + fileKey
}

// GetThumbnailURL generates the public URL for a thumbnail
func (s *S3Client) GetThumbnailURL(fileKey string) string {
	return "https://" + s.thumbnailBucket + ".s3." + s.region + ".amazonaws.com/" + fileKey
}
