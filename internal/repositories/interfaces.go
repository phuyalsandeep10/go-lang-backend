package repositories

import (
	"context"
	"time"

	"homeinsight-properties/internal/models"
)

type PropertyRepository interface {
	FindByID(ctx context.Context, id string) (*models.Property, error)
	FindByAddress(ctx context.Context, street, city, state, zip string) (*models.Property, error)
	FindWithPagination(ctx context.Context, offset, limit int) ([]models.Property, int64, error)
	Create(ctx context.Context, property *models.Property) error
	Update(ctx context.Context, property *models.Property) error
	Delete(ctx context.Context, id string) error
	FindAll(ctx context.Context) ([]models.Property, error)
}

type PropertyCache interface {
	GetProperty(ctx context.Context, key string) (*models.Property, error)
	SetProperty(ctx context.Context, key string, property *models.Property, expiration time.Duration) error
	GetSearchKey(ctx context.Context, key string) (string, error)
	SetSearchKey(ctx context.Context, key, propertyID string, expiration time.Duration) error
	AddCacheKeyToPropertySet(ctx context.Context, propertyID, cacheKey string) error
	InvalidatePropertyCacheKeys(ctx context.Context, propertyID string) error
	Delete(ctx context.Context, key string) error
	ClearAll(ctx context.Context) error
}



// UserRepository defines the interface for user data operations
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
}
