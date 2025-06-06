package handlers

import (
	"net/http"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/services"
	"github.com/gin-gonic/gin"
)

type PropertyHandler struct {
	service *services.PropertyService
}

func NewPropertyHandler() *PropertyHandler {
	return &PropertyHandler{
		service: services.NewPropertyService(),
	}
}

// CreateProperty handles POST /properties
func (h *PropertyHandler) CreateProperty(c *gin.Context) {
	var property models.Property
	if err := c.ShouldBindJSON(&property); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if err := h.service.CreateProperty(&property); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, property)
}

// ListProperties handles GET /properties
func (h *PropertyHandler) ListProperties(c *gin.Context) {
	properties, err := h.service.GetAllProperties()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, properties)
}
