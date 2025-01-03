package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ClerkID     string    `json:"clerk_id" db:"clerk_id"`
	Email       string    `json:"email" db:"email"`
	Name        string    `json:"name" db:"name"`
	DisplayName string    `json:"display_name" db:"display_name"`
	City        string    `json:"city" db:"city"`
	State       string    `json:"state" db:"state"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type CreateUserRequest struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	City        string `json:"city"`
	State       string `json:"state"`
	ClerkID     string `json:"clerk_id"`
}

type UpdateUserProfile struct {
	DisplayName *string `json:"display_name,omitempty"`
	City        *string `json:"city,omitempty"`
	State       *string `json:"state,omitempty"`
}
