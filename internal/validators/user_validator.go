package validators

import (
	"errors"
	"regexp"
	"homeinsight-properties/internal/models"
)

type userValidator struct{}

func NewUserValidator() UserValidator {
	return &userValidator{}
}

func (v *userValidator) ValidateRegister(user *models.User) error {
	if user.FullName == "" || user.Email == "" || user.Password == "" {
		return errors.New("full name, email, and password are required")
	}

	if len(user.FullName) < 2 || len(user.FullName) > 100 {
		return errors.New("full name must be between 2 and 100 characters")
	}

	if len(user.Password) < 6 || len(user.Password) > 100 {
		return errors.New("password must be between 6 and 100 characters")
	}

	if user.Phone != "" && len(user.Phone) > 15 {
		return errors.New("phone number exceeds maximum length of 15 characters")
	}

	if !isValidEmail(user.Email) {
		return errors.New("invalid email format")
	}

	if user.Phone != "" && !isValidPhone(user.Phone) {
		return errors.New("invalid phone format")
	}

	return nil
}

func (v *userValidator) ValidateLogin(email, password string) error {
	if email == "" || password == "" {
		return errors.New("email and password are required")
	}

	if !isValidEmail(email) {
		return errors.New("invalid email format")
	}

	if len(password) < 6 || len(password) > 100 {
		return errors.New("password must be between 6 and 100 characters")
	}

	return nil
}

func isValidEmail(email string) bool {
	regex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return regex.MatchString(email)
}

func isValidPhone(phone string) bool {
	regex := regexp.MustCompile(`^(\+\d{1,3}[- ]?)?\d{10}$`)
	return regex.MatchString(phone)
}
