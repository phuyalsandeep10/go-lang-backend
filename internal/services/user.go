package services

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/auth"
	"homeinsight-properties/pkg/config"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
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

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %v", err)
	}

	// Generate user ID
	user.ID = uuid.New().String()

	// Insert user into database
	query := "INSERT INTO users (id, full_name, email, phone, password) VALUES (?, ?, ?, ?, ?)"
	_, err = s.db.Exec(query, user.ID, user.FullName, user.Email, user.Phone, hashedPassword)
	if err != nil {
		return "", fmt.Errorf("failed to register user: %v", err)
	}

	// Generate JWT
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to load config: %v", err)
	}
	token, err := auth.GenerateJWT(user.ID, user.FullName, user.Email, user.Phone, cfg.JWT.Secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	return token, nil
}

func (s *UserService) Login(email, password string) (string, error) {
	var user models.User
	var hashedPassword string
	query := "SELECT id, full_name, email, phone, password FROM users WHERE email = ?"
	err := s.db.QueryRow(query, email).Scan(&user.ID, &user.FullName, &user.Email, &user.Phone, &hashedPassword)
	if err == sql.ErrNoRows {
		return "", errors.New("invalid email or password")
	}
	if err != nil {
		return "", fmt.Errorf("failed to query user: %v", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return "", errors.New("invalid email or password")
	}

	// Generate JWT
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to load config: %v", err)
	}
	token, err := auth.GenerateJWT(user.ID, user.FullName, user.Email, user.Phone, cfg.JWT.Secret)
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
