package server

import (
	"cource-api/internal/handlers"
	"cource-api/internal/middleware"
)

// RegisterRoutes configures all the routes for the application
func (s *FiberServer) RegisterRoutes() {
	// Public routes
	api := s.App.Group("/api")
	v1 := api.Group("/v1")

	// Auth routes
	auth := v1.Group("/auth")
	auth.Post("/register", handlers.HandleRegister(s.UserRepo, s.OTPRepo))
	auth.Post("/login", handlers.HandleLogin(s.UserRepo))
	// auth.Post("/otp/generate", handlers.HandleGenerateOTP(s.OTPRepo))
	auth.Post("/otp/verify", handlers.HandleVerifyOTP(s.OTPRepo, s.UserRepo))

	// Protected routes
	protected := v1.Group("/", middleware.AuthMiddleware())

	// User routes
	users := protected.Group("/users")
	users.Get("/me", handlers.HandleGetCurrentUser(s.UserRepo))
	users.Put("/me", handlers.HandleUpdateCurrentUser(s.UserRepo))

	// Course routes
	courses := protected.Group("/courses")
	courses.Get("/", handlers.HandleListCourses(s.CourseRepo))
	courses.Post("/", middleware.RequireRole("admin"), handlers.HandleCreateCourse(s.CourseRepo))
	courses.Get("/:id", handlers.HandleGetCourse(s.CourseRepo))
	courses.Put("/:id", middleware.RequireRole("admin"), handlers.HandleUpdateCourse(s.CourseRepo))
	courses.Delete("/:id", middleware.RequireRole("admin"), handlers.HandleDeleteCourse(s.CourseRepo))

	//aws s3 routes
	awsRoutes := protected.Group("/s3")
	awsRoutes.Post("/generate-video-url", handlers.HandleVideoGeneratePresignedURL())
	awsRoutes.Post("/generate-thumbnail-url", handlers.HandleThumbnailGeneratePresignedURL())

	// Video routes
	videos := protected.Group("/videos")
	videos.Get("/", handlers.HandleListVideos(s.VideoRepo))
	videos.Post("/", middleware.RequireRole("admin"), handlers.HandleCreateVideo(s.VideoRepo, s.CourseRepo))
	videos.Post("/reorder/:id", middleware.RequireRole("admin"), handlers.HandleReorderVideos(s.CourseRepo))
	videos.Get("/:id", handlers.HandleGetVideo(s.VideoRepo))
	videos.Put("/:id", middleware.RequireRole("admin"), handlers.HandleUpdateVideo(s.VideoRepo, s.CourseRepo))
	videos.Delete("/:id", middleware.RequireRole("admin"), handlers.HandleDeleteVideo(s.VideoRepo, s.CourseRepo))
	videos.Post("/:id/watch", handlers.HandleUpdateWatchHistory(s.VideoRepo))
	videos.Get("/history", handlers.HandleGetWatchHistory(s.VideoRepo))

	// Payment routes
	payments := protected.Group("/payments")
	payments.Get("/", handlers.HandleListPayments(s.PaymentRepo))
	payments.Post("/", handlers.HandleCreatePayment(s.PaymentRepo))
	payments.Get("/:id", handlers.HandleGetPayment(s.PaymentRepo))
	payments.Get("/pricing", handlers.HandleGetRegionalPricing(s.PaymentRepo))

	// Subscription routes
	subscriptions := protected.Group("/subscriptions")
	subscriptions.Post("/", handlers.HandleCreateSubscription(s.SubscriptionRepo, s.ProductRepo))
	subscriptions.Get("/", handlers.HandleListSubscriptions(s.SubscriptionRepo))
	subscriptions.Get("/:id", handlers.HandleGetSubscription(s.SubscriptionRepo))
	subscriptions.Post("/:id/cancel", handlers.HandleCancelSubscription(s.SubscriptionRepo))
	subscriptions.Post("/:id/reactivate", handlers.HandleReactivateSubscription(s.SubscriptionRepo))
	subscriptions.Put("/:id/payment-method", handlers.HandleUpdatePaymentMethod(s.SubscriptionRepo))

	// Product routes (admin only)
	products := protected.Group("/products", middleware.RequireRole("admin"))
	products.Get("/", handlers.HandleListProducts(s.ProductRepo))
	products.Post("/", handlers.HandleCreateProduct(s.ProductRepo))
	products.Get("/:id", handlers.HandleGetProduct(s.ProductRepo))
	products.Put("/:id", handlers.HandleUpdateProduct(s.ProductRepo))
	products.Delete("/:id", handlers.HandleDeleteProduct(s.ProductRepo))
	products.Put("/:id/price", handlers.HandleUpdateProductPrice(s.ProductRepo))
	products.Put("/:id/status", handlers.HandleUpdateProductStatus(s.ProductRepo))

	// Stripe webhook (public route)
	v1.Post("/webhook/stripe", handlers.HandleStripeWebhook(s.PaymentRepo))

	// Admin routes
	admin := protected.Group("/admin", middleware.RequireRole("admin"))
	admin.Get("/users", handlers.HandleListUsers(s.UserRepo))
	admin.Get("/users/stats", handlers.HandleGetUserStats(s.UserRepo))
	admin.Put("/users/:id", handlers.HandleUpdateUser(s.UserRepo))
	admin.Delete("/users/:id", handlers.HandleDeleteUser(s.UserRepo))
	admin.Get("/courses", handlers.HandleAdminListCourses(s.CourseRepo))

	admin.Put("/pricing/:region", handlers.HandleUpdateRegionalPricing(s.PaymentRepo))
}
