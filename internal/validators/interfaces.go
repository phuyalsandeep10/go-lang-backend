package validators

import (
	"homeinsight-properties/internal/models"
)

type PropertyValidator interface {
	ValidateCreate(property *models.Property) error
	ValidateUpdate(property *models.Property) error
	ValidateSearch(req *models.SearchRequest) error
}


type UserValidator interface {
	ValidateRegister(user *models.User) error
	ValidateLogin(email, password string) error
}
