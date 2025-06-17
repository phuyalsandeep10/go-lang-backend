package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/auth"
	"homeinsight-properties/pkg/config"
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
	count, err := collection.CountDocuments(ctx, bson.M{"email": user.Email})
	if err != nil {
		return "", fmt.Errorf("failed to check email existence: %v", err)
	}
	if count > 0 {
		return "", errors.New("email already registered")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %v", err)
	}

	// Generate MongoDB ObjectID
	user.ID = primitive.NewObjectID()
	user.Password = string(hashedPassword)

	// Insert user into MongoDB
	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		return "", fmt.Errorf("failed to register user: %v", err)
	}

	// Generate JWT
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to load config: %v", err)
	}
	token, err := auth.GenerateJWT(user.ID.Hex(), user.FullName, user.Email, user.Phone, cfg.JWT.Secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	return token, nil
}

func (s *UserService) Login(email, password string) (string, error) {
	ctx := context.Background()
	collection := s.db.Collection("users")

	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return "", errors.New("invalid email or password")
	}
	if err != nil {
		return "", fmt.Errorf("failed to query user: %v", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid email or password")
	}

	// Generate JWT
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to load config: %v", err)
	}
	token, err := auth.GenerateJWT(user.ID.Hex(), user.FullName, user.Email, user.Phone, cfg.JWT.Secret)
	if err != nil {
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
