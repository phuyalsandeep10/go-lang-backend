package services

import (
	"context"
	"fmt"
	"homeinsight-properties/internal/auth"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/repositories"
	"homeinsight-properties/internal/validators"
	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/metrics"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
    repo      repositories.UserRepository
    validator validators.UserValidator
    cfg       *config.Config
}

func NewUserService(repo repositories.UserRepository, validator validators.UserValidator) *UserService {
    cfg, err := config.LoadConfig("configs/config.yaml")
    if err != nil {
        cfg = &config.Config{} // Fallback to empty config
    }
    return &UserService{
        repo:      repo,
        validator: validator,
        cfg:       cfg,
    }
}

func (s *UserService) Register(user *models.User) (*auth.TokenDetails, error) {
    // Validate user input
    if err := s.validator.ValidateRegister(user); err != nil {
        return nil, err
    }

    // Check if email already exists
    ctx := context.Background()
    if existingUser, err := s.repo.FindByEmail(ctx, user.Email); err == nil && existingUser != nil {
        return nil, fmt.Errorf("email already registered")
    } else if err != nil && err != mongo.ErrNoDocuments {
        return nil, fmt.Errorf("failed to check email existence: %v", err)
    }

    // Hash the password
    start := time.Now()
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
    duration := time.Since(start).Seconds()
    metrics.MongoOperationDuration.WithLabelValues("hash_password", "").Observe(duration)
    if err != nil {
        metrics.MongoErrorsTotal.WithLabelValues("hash_password", "").Inc()
        return nil, fmt.Errorf("failed to hash password: %v", err)
    }

    user.ID = primitive.NewObjectID()
    user.Password = string(hashedPassword)

    // Create user in the database
    if err := s.repo.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("failed to register user: %v", err)
    }

    // Generate JWT
    start = time.Now()
    tokenDetails, err := auth.GenerateJWT(user.ID.Hex(), user.FullName, user.Email, user.Phone, s.cfg.JWT.Secret)
    duration = time.Since(start).Seconds()
    metrics.MongoOperationDuration.WithLabelValues("generate_jwt", "").Observe(duration)
    if err != nil {
        metrics.MongoErrorsTotal.WithLabelValues("generate_jwt", "").Inc()
        return nil, fmt.Errorf("failed to generate token: %v", err)
    }

    return tokenDetails, nil
}

func (s *UserService) Login(email, password string) (*auth.TokenDetails, error) {
    // Validate login input
    if err := s.validator.ValidateLogin(email, password); err != nil {
        return nil, err
    }

    // Find user by email
    ctx := context.Background()
    user, err := s.repo.FindByEmail(ctx, email)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, fmt.Errorf("invalid email or password")
        }
        return nil, fmt.Errorf("failed to query user: %v", err)
    }

    // Verify password
    start := time.Now()
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
        duration := time.Since(start).Seconds()
        metrics.MongoOperationDuration.WithLabelValues("verify_password", "").Observe(duration)
        metrics.MongoErrorsTotal.WithLabelValues("verify_password", "").Inc()
        return nil, fmt.Errorf("invalid email or password")
    }
    duration := time.Since(start).Seconds()
    metrics.MongoOperationDuration.WithLabelValues("verify_password", "").Observe(duration)

    // Generate JWT
    start = time.Now()
    tokenDetails, err := auth.GenerateJWT(user.ID.Hex(), user.FullName, user.Email, user.Phone, s.cfg.JWT.Secret)
    duration = time.Since(start).Seconds()
    metrics.MongoOperationDuration.WithLabelValues("generate_jwt", "").Observe(duration)
    if err != nil {
        metrics.MongoErrorsTotal.WithLabelValues("generate_jwt", "").Inc()
        return nil, fmt.Errorf("failed to generate token: %v", err)
    }

    return tokenDetails, nil
}
