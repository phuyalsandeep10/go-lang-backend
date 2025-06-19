// handlers/property_handler.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/services"
)

type PropertyHandler struct {
	propertyService *services.PropertyService
}

func NewPropertyHandler(propertyService *services.PropertyService) *PropertyHandler {
	return &PropertyHandler{propertyService: propertyService}
}

// GetProperties godoc
// @Summary Get all properties with pagination
// @Description Get a paginated list of all properties
// @Tags Properties
// @Accept json
// @Produce json
// @Param offset query int false "Offset for pagination" default(0)
// @Param limit query int false "Limit for pagination" default(10)
// @Security BearerAuth
// @Success 200 {object} models.PaginatedPropertiesResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /properties [get]
func (h *PropertyHandler) GetProperties(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	response, err := h.propertyService.GetPropertiesWithPagination(c, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

// SearchProperty godoc
// @Summary Search for a specific property
// @Description Search for properties based on query string
// @Tags Properties
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Security BearerAuth
// @Success 200 {object} models.Property
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /properties/property-search [get]
func (h *PropertyHandler) SearchProperty(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	req := &models.SearchRequest{Search: query}
	property, err := h.propertyService.SearchSpecificProperty(c, req)
	if err != nil {
		if err.Error() == "property not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, property)
}

// GetPropertyByID godoc
// @Summary Get property by ID
// @Description Get a single property by its ID
// @Tags Properties
// @Accept json
// @Produce json
// @Param id path string true "Property ID"
// @Security BearerAuth
// @Success 200 {object} models.Property
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /properties/{id} [get]
func (h *PropertyHandler) GetPropertyByID(c *gin.Context) {
	id := c.Param("id")
	property, err := h.propertyService.GetPropertyByID(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
		return
	}
	c.JSON(http.StatusOK, property)
}

// CreateProperty godoc
// @Summary Create a new property
// @Description Create a new property record
// @Tags Properties
// @Accept json
// @Produce json
// @Param property body models.Property true "Property data"
// @Security BearerAuth
// @Success 201 {object} models.Property
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /properties [post]
func (h *PropertyHandler) CreateProperty(c *gin.Context) {
	var property models.Property
	if err := c.ShouldBindJSON(&property); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.propertyService.CreateProperty(c, &property); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, property)
}

func (h *PropertyHandler) UpdateProperty(c *gin.Context) {
	var property models.Property
	if err := c.ShouldBindJSON(&property); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.propertyService.UpdateProperty(c, &property); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, property)
}

func (h *PropertyHandler) DeleteProperty(c *gin.Context) {
	id := c.Param("id")
	if err := h.propertyService.DeleteProperty(c, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
