package model

import "time"

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey;column:id"`
	Email        string    `json:"email" gorm:"uniqueIndex;column:email"`
	PasswordHash string    `json:"-" gorm:"column:password_hash"`
	Role         string    `json:"role" gorm:"column:role;default:'reader'"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type UpdateProfileRequest struct {
	Email string `json:"email" binding:"required,email"`
}
