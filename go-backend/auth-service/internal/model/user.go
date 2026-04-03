// Package model defines the User struct and related types for AuthService.
package model

import "time"

// User represents a registered user. Password is bcrypt-hashed.
type User struct {
	ID        int       `json:"id" bson:"_id"`
	Email     string    `json:"email" bson:"email"`
	Password  string    `json:"password,omitempty" bson:"password"`
	FirstName string    `json:"first_name" bson:"first_name"`
	LastName  string    `json:"last_name" bson:"last_name"`
	Phone     string    `json:"phone" bson:"phone"`
	AvatarURL string    `json:"avatar_url" bson:"avatar_url"`
	Role      string    `json:"role" bson:"role"` // "admin" or "customer"
	IsActive  bool      `json:"is_active" bson:"is_active"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// RegisterRequest is the payload for POST /auth/register
type RegisterRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Phone           string `json:"phone"`
}

// UpdateProfileRequest is the payload for PUT /auth/profile
type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password"` // optional
}

// LoginRequest is the payload for POST /auth/login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is returned after successful login/register.
// Contains the JWT and basic user profile.
type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"` // seconds
	User      struct {
		ID        int    `json:"id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
	} `json:"user"`
}
