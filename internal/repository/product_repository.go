package repository

import (
	"context"
	"errors"
	"time"

	"cource-api/internal/database"
	"cource-api/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductRepository struct {
	collection *mongo.Collection
}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{
		collection: database.Products,
	}
}

// Create creates a new product
func (r *ProductRepository) Create(ctx context.Context, product *models.Product) error {
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, product)
	if err != nil {
		return err
	}

	product.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID finds a product by ID
func (r *ProductRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Product, error) {
	var product models.Product
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&product)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

// GetByProductID finds a product by its external product ID
func (r *ProductRepository) GetByProductID(ctx context.Context, productID string) (*models.Product, error) {
	var product models.Product
	err := r.collection.FindOne(ctx, bson.M{"product_id": productID}).Decode(&product)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

// List returns a list of products with pagination
func (r *ProductRepository) List(ctx context.Context, page, limit int64) ([]*models.Product, int64, error) {
	skip := (page - 1) * limit

	// Get total count
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	// Find products with pagination
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var products []*models.Product
	if err = cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// Update updates a product
func (r *ProductRepository) Update(ctx context.Context, product *models.Product) error {
	product.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"product_id":     product.ProductID,
			"interval":       product.Interval,
			"currency":       product.Currency,
			"status":         product.Status,
			"price":          product.Price,
			"original_price": product.OriginalPrice,
			"iap_product_id": product.IAPProductID,
			"price_id":       product.PriceID,
			"type":           product.Type,
			"trial_days":     product.TrialDays,
			"updated_at":     product.UpdatedAt,
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": product.ID},
		update,
	)
	return err
}

// Delete deletes a product
func (r *ProductRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// ListActive returns a list of active products
func (r *ProductRepository) ListActive(ctx context.Context) ([]*models.Product, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []*models.Product
	if err = cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}

// UpdatePrice updates a product's price
func (r *ProductRepository) UpdatePrice(ctx context.Context, id primitive.ObjectID, price, originalPrice float64) error {
	update := bson.M{
		"$set": bson.M{
			"price":          price,
			"original_price": originalPrice,
			"updated_at":     time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		update,
	)
	return err
}

// UpdateStatus updates a product's status
func (r *ProductRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status bool) error {
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		update,
	)
	return err
}
