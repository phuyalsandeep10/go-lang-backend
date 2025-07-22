
package handlers

import (
	"net/http"
	"strconv"

	"homeinsight-properties/internal/errors"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/services"
	"homeinsight-properties/internal/utils"
	"homeinsight-properties/pkg/logger"

	"github.com/gin-gonic/gin"
)

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
		appErr := errors.NewAppError(
			"invalid offset parameter",
			errors.MsgInvalidParameters,
			errors.ErrCodeInvalidParameters,
			http.StatusBadRequest,
			err,
		)
		logger.GlobalLogger.Errorf("Invalid offset: value=%s, error=%v", offsetStr, appErr.TechnicalMessage)
		c.Error(appErr)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		appErr := errors.NewAppError(
			"invalid limit parameter",
			errors.MsgInvalidParameters,
			errors.ErrCodeInvalidParameters,
			http.StatusBadRequest,
			err,
		)
		logger.GlobalLogger.Errorf("Invalid limit: value=%s, error=%v", limitStr, appErr.TechnicalMessage)
		c.Error(appErr)
		return
	}

	response, err := h.searchService.ListProperties(c, offset, limit, "/api/properties", c.Request.URL.Query())
	if err != nil {
		c.Error(utils.LogAndMapError(c, err, "get properties",
			"offset", offset,
			"limit", limit))
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *PropertyHandler) SearchProperty(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		appErr := errors.NewAppError(
			"query parameter missing",
			"Search query is required",
			errors.ErrCodeInvalidParameters,
			http.StatusBadRequest,
			nil,
		)
		logger.GlobalLogger.Errorf("Missing query parameter: path=%s", c.Request.URL.Path)
		c.Error(appErr)
		return
	}
	if len(query) > 100 {
		appErr := errors.NewAppError(
			"query parameter too long",
			"Search query exceeds maximum length of 100 characters",
			errors.ErrCodeInvalidParameters,
			http.StatusBadRequest,
			nil,
		)
		logger.GlobalLogger.Errorf("Query too long: query=%s", query)
		c.Error(appErr)
		return
	}

	req := &models.SearchRequest{Search: query}
	property, err := h.searchService.SearchSpecificProperty(c, req)
	if err != nil {
		c.Error(utils.LogAndMapError(c, err, "search specific property", "query", query))
		return
	}
	c.JSON(http.StatusOK, property)
}

func (h *PropertyHandler) GetPropertyByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		appErr := errors.NewAppError(
			"id parameter missing",
			"Property ID is required",
			errors.ErrCodeInvalidParameters,
			http.StatusBadRequest,
			nil,
		)
		logger.GlobalLogger.Errorf("Missing ID parameter: path=%s", c.Request.URL.Path)
		c.Error(appErr)
		return
	}

	property, err := h.propertyService.GetPropertyByID(c, id)
	if err != nil {
		c.Error(utils.LogAndMapError(c, err, "get property by ID", "id", id))
		return
	}
	c.JSON(http.StatusOK, property)
}

func (h *PropertyHandler) CreateProperty(c *gin.Context) {
	var property models.Property
	if err := c.ShouldBindJSON(&property); err != nil {
		appErr := errors.NewAppError(
			"invalid request body",
			"The provided property data is invalid",
			errors.ErrCodeInvalidParameters,
			http.StatusBadRequest,
			err,
		)
		logger.GlobalLogger.Errorf("Invalid property data: error=%v", err)
		c.Error(appErr)
		return
	}

	if err := h.propertyService.CreateProperty(c, &property); err != nil {
		c.Error(utils.LogAndMapError(c, err, "create property"))
		return
	}
	c.JSON(http.StatusCreated, property)
}

func (h *PropertyHandler) UpdateProperty(c *gin.Context) {
	var property models.Property
	if err := c.ShouldBindJSON(&property); err != nil {
		appErr := errors.NewAppError(
			"invalid request body",
			"The provided property data is invalid",
			errors.ErrCodeInvalidParameters,
			http.StatusBadRequest,
			err,
		)
		logger.GlobalLogger.Errorf("Invalid property data: error=%v", err)
		c.Error(appErr)
		return
	}

	if err := h.propertyService.UpdateProperty(c, &property); err != nil {
		c.Error(utils.LogAndMapError(c, err, "update property"))
		return
	}
	c.JSON(http.StatusOK, property)
}

func (h *PropertyHandler) DeleteProperty(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		appErr := errors.NewAppError(
			"id parameter missing",
			"Property ID is required",
			errors.ErrCodeInvalidParameters,
			http.StatusBadRequest,
			nil,
		)
		logger.GlobalLogger.Errorf("Missing ID parameter: path=%s", c.Request.URL.Path)
		c.Error(appErr)
		return
	}

	if err := h.propertyService.DeleteProperty(c, id); err != nil {
		c.Error(utils.LogAndMapError(c, err, "delete property", "id", id))
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
