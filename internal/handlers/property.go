package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"homeinsight-properties/internal/models"
)

type PropertyHandler struct {
	db *sql.DB
}

func NewPropertyHandler(db *sql.DB) *PropertyHandler {
	return &PropertyHandler{db: db}
}

func (h *PropertyHandler) ListProperties(c *gin.Context) {
	var properties []models.Property
	rows, err := h.db.Query("SELECT id, name, description, price FROM properties")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Property
		var description sql.NullString // Handle NULL description
		if err := rows.Scan(&p.ID, &p.Name, &description, &p.Price); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		p.Description = description.String // Convert sql.NullString to string
		properties = append(properties, p)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, properties)
}

func (h *PropertyHandler) CreateProperty(c *gin.Context) {
	var p models.Property
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate a UUID for the ID if not provided
	if p.ID == "" {
		p.ID = uuid.New().String()
	}

	// Insert into database
	_, err := h.db.Exec(
		"INSERT INTO properties (id, name, description, price) VALUES (?, ?, ?, ?)",
		p.ID, p.Name, p.Description, p.Price,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, p)
}
