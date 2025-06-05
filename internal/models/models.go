package models

import (
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	Name         string             `bson:"name" json:"name"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	Role         string             `bson:"role" json:"role"`
	IsVerified   bool               `bson:"is_verified" json:"is_verified"`
	Subscription Subscription       `bson:"subscription" json:"subscription"`
	Blocked      bool               `bson:"blocked" json:"-"`
	CreatedAt    time.Time          `bson:"created_at" json:"-"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"-"`
}

// OTP represents a one-time password for verification
type OTP struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email     string             `bson:"email" json:"email"`
	Code      string             `bson:"code" json:"-"`
	Type      string             `bson:"type" json:"type"` // "registration" or "reset"
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	Used      bool               `bson:"used" json:"used"`
}

// VerifyPassword checks if the provided password matches the stored hash
func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id": u.ID,
			"email":   u.Email,
			"error":   err,
		}).Error("Password verification failed")
		return false
	}
	return true
}

// Subscription represents a user's subscription details
type Subscription struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID             primitive.ObjectID `bson:"user_id" json:"user_id"`
	ProductID          primitive.ObjectID `bson:"product_id" json:"product_id"`
	Status             string             `bson:"status" json:"status"` // active, canceled, expired, trial
	Plan               string             `bson:"plan" json:"plan"`     // monthly, yearly, etc.
	Region             string             `bson:"region" json:"region"`
	Currency           string             `bson:"currency" json:"currency"`
	Amount             float64            `bson:"amount" json:"amount"`
	CurrentPeriodStart time.Time          `bson:"current_period_start" json:"current_period_start"`
	CurrentPeriodEnd   time.Time          `bson:"current_period_end" json:"current_period_end"`
	CancelAtPeriodEnd  bool               `bson:"cancel_at_period_end" json:"cancel_at_period_end"`
	CanceledAt         *time.Time         `bson:"canceled_at,omitempty" json:"canceled_at,omitempty"`
	TrialStart         *time.Time         `bson:"trial_start,omitempty" json:"trial_start,omitempty"`
	TrialEnd           *time.Time         `bson:"trial_end,omitempty" json:"trial_end,omitempty"`
	PaymentMethodID    string             `bson:"payment_method_id" json:"payment_method_id"`
	CustomerID         string             `bson:"customer_id" json:"customer_id"`
	SubscriptionID     string             `bson:"subscription_id" json:"subscription_id"`
	LastPaymentStatus  string             `bson:"last_payment_status" json:"last_payment_status"`
	LastPaymentDate    *time.Time         `bson:"last_payment_date,omitempty" json:"last_payment_date,omitempty"`
	NextBillingDate    *time.Time         `bson:"next_billing_date,omitempty" json:"next_billing_date,omitempty"`
	AutoRenew          bool               `bson:"auto_renew" json:"auto_renew"`
	CreatedAt          time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt          time.Time          `bson:"updated_at" json:"updated_at"`
}

// Course represents a course in the system
type Course struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Title        string               `bson:"title" json:"title"`
	SubTitle     string               `bson:"subtitle" json:"subtitle"`
	Description  string               `bson:"description" json:"description"`
	ThumbnailURL string               `bson:"thumbnail_url" json:"thumbnail_url"`
	VideoOrder   []primitive.ObjectID `bson:"video_order" json:"video_order"` // Ordered array of video IDs
	IsPaid       bool                 `bson:"is_paid" json:"is_paid"`
	Skills       []string             `bson:"skills" json:"skills"`
	Author       string               `bson:"author" json:"author"`
	CreatedBy    primitive.ObjectID   `bson:"created_by" json:"created_by"`
	CreatedAt    time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time            `bson:"updated_at" json:"updated_at"`
}

// Product represents a subscription product in the system
type Product struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID     string             `bson:"product_id" json:"product_id"`         // External product ID (e.g., Stripe)
	Interval      string             `bson:"interval" json:"interval"`             // monthly, yearly, etc.
	Currency      string             `bson:"currency" json:"currency"`             // USD, EUR, etc.
	Status        bool               `bson:"status" json:"status"`                 // Active/Inactive
	Price         float64            `bson:"price" json:"price"`                   // Current price
	OriginalPrice float64            `bson:"original_price" json:"original_price"` // Original price (for discounts)
	IAPProductID  string             `bson:"iap_product_id" json:"iap_product_id"` // In-app purchase product ID
	PriceID       string             `bson:"price_id" json:"price_id"`             // External price ID (e.g., Stripe)
	Type          string             `bson:"type" json:"type"`                     // subscription, one_time, etc.
	TrialDays     int                `bson:"trial_days" json:"trial_days"`         // Number of trial days
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// Video represents a video in the system
type Video struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	URL         string             `bson:"url" json:"url"`
	Thumbnail   string             `bson:"thumbnail" json:"thumbnail"`
	Duration    int                `bson:"duration" json:"duration"`
	IsPaid      bool               `bson:"is_paid" json:"is_paid"`
	CourseID    primitive.ObjectID `bson:"course_id" json:"course_id"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

// WatchHistory represents a user's video watch history
type WatchHistory struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID          primitive.ObjectID `bson:"user_id" json:"user_id"`
	VideoID         primitive.ObjectID `bson:"video_id" json:"video_id"`
	LastWatchedAt   time.Time          `bson:"last_watched_at" json:"last_watched_at"`
	ProgressSeconds int                `bson:"progress_seconds" json:"progress_seconds"`
}

// Payment represents a payment transaction
type Payment struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        primitive.ObjectID `bson:"user_id" json:"user_id"`
	Gateway       string             `bson:"gateway" json:"gateway"`
	TransactionID string             `bson:"transaction_id" json:"transaction_id"`
	Amount        int                `bson:"amount" json:"amount"`
	Currency      string             `bson:"currency" json:"currency"`
	Region        string             `bson:"region" json:"region"`
	Status        string             `bson:"status" json:"status"`
	Timestamp     time.Time          `bson:"timestamp" json:"timestamp"`
}

// RegionalPricing represents pricing for different regions
type RegionalPricing struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RegionCode     string             `bson:"region_code" json:"region_code"`
	Currency       string             `bson:"currency" json:"currency"`
	MonthlyPrice   int                `bson:"monthly_price" json:"monthly_price"`
	YearlyPrice    int                `bson:"yearly_price" json:"yearly_price"`
	CurrencySymbol string             `bson:"currency_symbol" json:"currency_symbol"`
}
