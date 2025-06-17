package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type PropertyHandler struct {
	db              *mongo.Database
	propertyService *services.PropertyService
}

func NewPropertyHandler(db *mongo.Database) *PropertyHandler {
	return &PropertyHandler{
		db:              db,
		propertyService: services.NewPropertyService(),
	}
}

func (h *PropertyHandler) ListProperties(c *gin.Context) {
	// Parse pagination parameters
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "10")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		fmt.Printf("Invalid offset parameter: %s\n", offsetStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset parameter"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		fmt.Printf("Invalid limit parameter: %s\n", limitStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter (1-100)"})
		return
	}

	result, err := h.propertyService.GetPropertiesWithPagination(c, offset, limit)
	if err != nil {
		fmt.Printf("Error listing properties (offset=%d, limit=%d): %v\n", offset, limit, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PropertyHandler) GetProperty(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		fmt.Println("Property ID is empty")
		c.JSON(http.StatusBadRequest, gin.H{"error": "property ID is required"})
		return
	}

	property, err := h.propertyService.GetPropertyByID(c, id)
	if err != nil {
		if err.Error() == "property not found" {
			fmt.Printf("Property %s not found\n", id)
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}
		fmt.Printf("Error fetching property %s: %v\n", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to fetch property: %v", err)})
		return
	}

	c.JSON(http.StatusOK, property)
}

func (h *PropertyHandler) CreateProperty(c *gin.Context) {
	var p models.Property
	if err := c.ShouldBindJSON(&p); err != nil {
		fmt.Printf("Error binding JSON for property creation: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.propertyService.CreateProperty(c, &p); err != nil {
		fmt.Printf("Error creating property %s: %v\n", p.PropertyID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

func (h *PropertyHandler) UpdateProperty(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		fmt.Println("Property ID is empty for update")
		c.JSON(http.StatusBadRequest, gin.H{"error": "property ID is required"})
		return
	}

	var p models.Property
	if err := c.ShouldBindJSON(&p); err != nil {
		fmt.Printf("Error binding JSON for property update %s: %v\n", id, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.PropertyID = id

	if err := h.propertyService.UpdateProperty(c, &p); err != nil {
		if err.Error() == "property not found" {
			fmt.Printf("Property %s not found for update\n", id)
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}
		fmt.Printf("Error updating property %s: %v\n", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *PropertyHandler) DeleteProperty(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		fmt.Println("Property ID is empty for delete")
		c.JSON(http.StatusBadRequest, gin.H{"error": "property id is required"})
		return
	}

	if err := h.propertyService.DeleteProperty(c, id); err != nil {
		if err.Error() == "property not found" {
			fmt.Printf("Property %s not found\n", id)
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}
		fmt.Printf("Error deleting property %s: %v\n", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "property deleted successfully"})
}
