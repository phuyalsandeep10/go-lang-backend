package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/services"
	"homeinsight-properties/pkg/logger"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type PropertyHandler struct {
	propertyService *services.PropertyService
	searchService   *services.PropertySearchService
}

func NewPropertyHandler(propertyService *services.PropertyService, searchService *services.PropertySearchService) *PropertyHandler {
	return &PropertyHandler{
		propertyService: propertyService,
		searchService:   searchService,
	}
}

func (h *PropertyHandler) GetProperties(c *gin.Context) {
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "10")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be a non-negative integer"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer between 1 and 100"})
		return
	}

	response, err := h.searchService.GetPropertiesWithPagination(c, offset, limit, "/api/properties", c.Request.URL.Query())
	if err != nil {
		logger.Logger.Printf("Error fetching properties: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *PropertyHandler) SearchProperty(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}
	if len(query) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' exceeds maximum length"})
		return
	}

	req := &models.SearchRequest{Search: query}
	property, err := h.searchService.SearchSpecificProperty(c, req)
	if err != nil {
		logger.Logger.Printf("Error searching property: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, property)
}

func (h *PropertyHandler) GetPropertyByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter is required"})
		return
	}

	property, err := h.propertyService.GetPropertyByID(c, id)
	if err != nil {
		logger.Logger.Printf("Error fetching property by ID: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
		return
	}
	c.JSON(http.StatusOK, property)
}

func (h *PropertyHandler) CreateProperty(c *gin.Context) {
	var property models.Property
	if err := c.ShouldBindJSON(&property); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.propertyService.CreateProperty(c, &property); err != nil {
		logger.Logger.Printf("Error creating property: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, property)
}

func (h *PropertyHandler) UpdateProperty(c *gin.Context) {
	var property models.Property
	if err := c.ShouldBindJSON(&property); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.propertyService.UpdateProperty(c, &property); err != nil {
		logger.Logger.Printf("Error updating property: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, property)
}

func (h *PropertyHandler) DeleteProperty(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter is required"})
		return
	}

	if err := h.propertyService.DeleteProperty(c, id); err != nil {
		logger.Logger.Printf("Error deleting property: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
