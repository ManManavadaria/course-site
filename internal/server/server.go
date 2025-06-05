package server

import (
	"cource-api/internal/config"
	"cource-api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

type FiberServer struct {
	App              *fiber.App
	UserRepo         *repository.UserRepository
	CourseRepo       *repository.CourseRepository
	VideoRepo        *repository.VideoRepository
	PaymentRepo      *repository.PaymentRepository
	OTPRepo          *repository.OTPRepository
	SubscriptionRepo *repository.SubscriptionRepository
	ProductRepo      *repository.ProductRepository
}

func New(
	userRepo *repository.UserRepository,
	courseRepo *repository.CourseRepository,
	videoRepo *repository.VideoRepository,
	paymentRepo *repository.PaymentRepository,
	otpRepo *repository.OTPRepository,
	subscriptionRepo *repository.SubscriptionRepository,
	productRepo *repository.ProductRepository,
) *FiberServer {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	app.Use(logger.New())
	app.Use(cors.New())

	return &FiberServer{
		App:              app,
		UserRepo:         userRepo,
		CourseRepo:       courseRepo,
		VideoRepo:        videoRepo,
		PaymentRepo:      paymentRepo,
		OTPRepo:          otpRepo,
		SubscriptionRepo: subscriptionRepo,
		ProductRepo:      productRepo,
	}
}

func (s *FiberServer) Listen() error {
	s.RegisterRoutes()
	return s.App.Listen(":" + config.AppConfig.ServerPort)
}
