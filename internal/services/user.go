package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"homeinsight-properties/pkg/auth"
	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/metrics" // Import metrics package

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"homeinsight-properties/internal/models"
)

type UserService struct {
	db *mongo.Database
}

func NewUserService(db *mongo.Database) *UserService {
	return &UserService{db: db}
}

func (s *UserService) Register(user *models.User) (string, error) {
	// Validate required fields
	if user.FullName == "" || user.Email == "" || user.Password == "" {
		return "", errors.New("full name, email, and password are required")
	}

	// Validate email format
	if !isValidEmail(user.Email) {
		return "", errors.New("invalid email format")
	}

	// Validate phone format (if provided)
	if user.Phone != "" && !isValidPhone(user.Phone) {
		return "", errors.New("invalid phone format")
	}

	// Check if email already exists
	ctx := context.Background()
	collection := s.db.Collection("users")
	start := time.Now()
	count, err := collection.CountDocuments(ctx, bson.M{"email": user.Email})
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("count_documents", "users").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("count_documents", "users").Inc()
		return "", fmt.Errorf("failed to check email existence: %v", err)
	}
	if count > 0 {
		return "", errors.New("email already registered")
	}

	// Hash the password
	start = time.Now()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("hash_password", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("hash_password", "").Inc()
		return "", fmt.Errorf("failed to hash password: %v", err)
	}

	// Generate MongoDB ObjectID
	user.ID = primitive.NewObjectID()
	user.Password = string(hashedPassword)

	// Insert user into MongoDB
	start = time.Now()
	_, err = collection.InsertOne(ctx, user)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("insert", "users").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("insert", "users").Inc()
		return "", fmt.Errorf("failed to register user: %v", err)
	}

	// Generate JWT
	start = time.Now()
	cfg, err := config.LoadConfig("configs/config.yaml")
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("load_config", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("load_config", "").Inc()
		return "", fmt.Errorf("failed to load config: %v", err)
	}
	start = time.Now()
	token, err := auth.GenerateJWT(user.ID.Hex(), user.FullName, user.Email, user.Phone, cfg.JWT.Secret)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("generate_jwt", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("generate_jwt", "").Inc()
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	return token, nil
}

func (s *UserService) Login(email, password string) (string, error) {
	ctx := context.Background()
	collection := s.db.Collection("users")

	var user models.User
	start := time.Now()
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("find_one", "users").Observe(duration)
	if err == mongo.ErrNoDocuments {
		metrics.MongoErrorsTotal.WithLabelValues("find_one", "users").Inc()
		return "", errors.New("invalid email or password")
	}
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("find_one", "users").Inc()
		return "", fmt.Errorf("failed to query user: %v", err)
	}

	// Verify password
	start = time.Now()
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		duration = time.Since(start).Seconds()
		metrics.MongoOperationDuration.WithLabelValues("verify_password", "").Observe(duration)
		metrics.MongoErrorsTotal.WithLabelValues("verify_password", "").Inc()
		return "", errors.New("invalid email or password")
	}
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("verify_password", "").Observe(duration)

	// Generate JWT
	start = time.Now()
	cfg, err := config.LoadConfig("configs/config.yaml")
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("load_config", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("load_config", "").Inc()
		return "", fmt.Errorf("failed to load config: %v", err)
	}
	start = time.Now()
	token, err := auth.GenerateJWT(user.ID.Hex(), user.FullName, user.Email, user.Phone, cfg.JWT.Secret)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("generate_jwt", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("generate_jwt", "").Inc()
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	return token, nil
}

// isValidEmail checks if the email matches a basic email format
func isValidEmail(email string) bool {
	regex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return regex.MatchString(email)
}

// isValidPhone checks if the phone number matches a basic format (e.g., +1234567890 or 123-456-7890)
func isValidPhone(phone string) bool {
	regex := regexp.MustCompile(`^(\+\d{1,3}[- ]?)?\d{10}$`)
	return regex.MatchString(phone)
}
