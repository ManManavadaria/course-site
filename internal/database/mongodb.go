package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client          *mongo.Client
	database        *mongo.Database
	Users           *mongo.Collection
	Courses         *mongo.Collection
	Videos          *mongo.Collection
	WatchHistory    *mongo.Collection
	Payments        *mongo.Collection
	RegionalPricing *mongo.Collection
	OTPs            *mongo.Collection
	Subscriptions   *mongo.Collection
	Products        *mongo.Collection
)

// Connect establishes a connection to MongoDB
func Connect(uri string, dbName string) error {
	fmt.Println("URI => ", uri)
	fmt.Println("DB Name => ", dbName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	var err error

	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		fmt.Println("Connection error => ", err)
		return err
	}

	// Ping the database
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		fmt.Println("Ping error => ", err)
		return err
	}

	database = client.Database(dbName)

	// Initialize collections
	Users = database.Collection("users")
	Courses = database.Collection("courses")
	Videos = database.Collection("videos")
	WatchHistory = database.Collection("watch_history")
	Payments = database.Collection("payments")
	RegionalPricing = database.Collection("regional_pricing")
	OTPs = database.Collection("otps")
	Subscriptions = database.Collection("subscriptions")
	Products = database.Collection("products")

	// Create indexes
	if err := createIndexes(); err != nil {
		fmt.Println("Create indexes error => ", err)
		return err
	}

	log.Println("Connected to MongoDB!")
	return nil
}

// createIndexes creates necessary indexes for the collections
func createIndexes() error {
	ctx := context.Background()

	// Users collection indexes
	_, err := Users.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "role", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "subscription.status", Value: 1}},
		},
	})
	if err != nil {
		return err
	}

	// OTPs collection indexes
	_, err = OTPs.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "email", Value: 1},
				{Key: "type", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	})
	if err != nil {
		return err
	}

	// WatchHistory collection indexes
	_, err = WatchHistory.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "video_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return err
	}

	// RegionalPricing collection indexes
	_, err = RegionalPricing.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "region_code", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return err
	}

	// Subscriptions collection indexes
	_, err = Subscriptions.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "current_period_end", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "subscription_id", Value: 1}},
		},
	})
	if err != nil {
		return err
	}

	// Products collection indexes
	_, err = Products.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "product_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// Disconnect closes the MongoDB connection
func Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Disconnect(ctx); err != nil {
		return err
	}

	log.Println("Disconnected from MongoDB!")
	return nil
}
