package main

import (
	"cource-api/internal/config"
	"cource-api/internal/database"
	"cource-api/internal/repository"
	"cource-api/internal/server"
	"log"
	"os"
)

func main() {
	// Load configuration
	if err := config.Load(); err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize MongoDB connection
	if err := database.Connect(config.AppConfig.MongoURI, config.AppConfig.DatabaseName); err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer database.Disconnect()

	// Initialize repositories
	userRepo := repository.NewUserRepository()
	videoRepo := repository.NewVideoRepository()
	courseRepo := repository.NewCourseRepository(videoRepo)
	paymentRepo := repository.NewPaymentRepository()
	otpRepo := repository.NewOTPRepository()
	subscriptionRepo := repository.NewSubscriptionRepository()
	productRepo := repository.NewProductRepository()

	// Initialize and start server
	srv := server.New(
		userRepo,
		courseRepo,
		videoRepo,
		paymentRepo,
		otpRepo,
		subscriptionRepo,
		productRepo,
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	srv.Listen()
}
