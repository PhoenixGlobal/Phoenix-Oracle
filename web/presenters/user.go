package presenters

import (
	"time"

	"PhoenixOracle/db/models"
)

type UserResource struct {
	JAID
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

func (r UserResource) GetName() string {
	return "users"
}

func NewUserResource(u models.User) *UserResource {
	return &UserResource{
		JAID:      NewJAID(u.Email),
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}
