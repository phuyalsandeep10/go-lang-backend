// internal/models/property.go
package models

type Property struct {
	ID          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	Description string  `json:"description" db:"description"`
	Price       float64 `json:"price" db:"price"`
}

type PaginationMeta struct {
	Total  int64   `json:"total"`
	Offset int     `json:"offset"`
	Limit  int     `json:"limit"`
	Next   *string `json:"next,omitempty"`
	Prev   *string `json:"prev,omitempty"`
}

type PaginatedPropertiesResponse struct {
	Data []Property     `json:"data"`
	Meta PaginationMeta `json:"meta"`
}
