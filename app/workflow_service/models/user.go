package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	Username  string     `db:"username" json:"username"`
	Email     string     `db:"email" json:"email"`
	Password  string     `db:"password" json:"password"`
	FirstName *string    `db:"first_name" json:"first_name"`
	LastName  *string    `db:"last_name" json:"last_name"`
	CreatedAt *time.Time `db:"created_at" json:"created_at"`
}

// user persistence methods to be implemeted
