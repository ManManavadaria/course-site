package handlers

import (
	"cource-api/internal/models"
	"cource-api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HandleListProducts returns a paginated list of products
func HandleListProducts(repo *repository.ProductRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page := c.QueryInt("page", 1)
		limit := c.QueryInt("limit", 10)

		products, total, err := repo.List(c.Context(), int64(page), int64(limit))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to list products")
		}

		return c.JSON(fiber.Map{
			"products": products,
			"total":    total,
			"page":     page,
			"limit":    limit,
		})
	}
}

// HandleCreateProduct creates a new product
func HandleCreateProduct(repo *repository.ProductRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var product models.Product
		if err := c.BodyParser(&product); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		if err := repo.Create(c.Context(), &product); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create product")
		}

		return c.Status(fiber.StatusCreated).JSON(product)
	}
}

// HandleGetProduct retrieves a product by ID
func HandleGetProduct(repo *repository.ProductRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid product ID")
		}

		product, err := repo.GetByID(c.Context(), objectID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "Product not found")
		}

		return c.JSON(product)
	}
}

// HandleUpdateProduct updates an existing product
func HandleUpdateProduct(repo *repository.ProductRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid product ID")
		}

		var product models.Product
		if err := c.BodyParser(&product); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		product.ID = objectID
		if err := repo.Update(c.Context(), &product); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update product")
		}

		return c.JSON(product)
	}
}

// HandleDeleteProduct deletes a product
func HandleDeleteProduct(repo *repository.ProductRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid product ID")
		}

		if err := repo.Delete(c.Context(), objectID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete product")
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// HandleUpdateProductPrice updates a product's price
func HandleUpdateProductPrice(repo *repository.ProductRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid product ID")
		}

		var request struct {
			Price         float64 `json:"price"`
			OriginalPrice float64 `json:"original_price"`
		}
		if err := c.BodyParser(&request); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		if err := repo.UpdatePrice(c.Context(), objectID, request.Price, request.OriginalPrice); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update product price")
		}

		return c.SendStatus(fiber.StatusOK)
	}
}

// HandleUpdateProductStatus updates a product's status
func HandleUpdateProductStatus(repo *repository.ProductRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid product ID")
		}

		var request struct {
			Status bool `json:"status"`
		}
		if err := c.BodyParser(&request); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		if err := repo.UpdateStatus(c.Context(), objectID, request.Status); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update product status")
		}

		return c.SendStatus(fiber.StatusOK)
	}
}
