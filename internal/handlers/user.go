package handlers

import (
	"net/http"
	"strings"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(db *mongo.Database) *UserHandler {
	return &UserHandler{
		userService: services.NewUserService(db),
	}
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=100" example:"John Doe"`
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Phone    string `json:"phone" binding:"omitempty,max=15" example:"123-456-7890"`
	Password string `json:"password" binding:"required,min=6,max=100" example:"password123"`
}

// TokenResponse represents the token response
type TokenResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param user body RegisterRequest true "User registration data"
// @Success 201 {object} TokenResponse
// @Failure 400 {object} map[string]string
// @Router /register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	user := models.User{
		FullName: strings.TrimSpace(req.FullName),
		Email:    strings.TrimSpace(req.Email),
		Phone:    strings.TrimSpace(req.Phone),
		Password: req.Password,
	}

	token, err := h.userService.Register(&user)
	if err != nil {
		if err.Error() == "email already registered" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
}

// Login godoc
// @Summary Login user
// @Description Authenticate user and return JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var creds LoginRequest
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
		return
	}

	creds.Email = strings.TrimSpace(creds.Email)
	token, err := h.userService.Login(creds.Email, creds.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
