// Package jwt provides shared JWT token generation and validation
// used across all microservices in the e-commerce platform.
package jwt

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT payload structure.
// All services share this same claims format.
type Claims struct {
	UserID    int    `json:"userId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	jwt.RegisteredClaims
}

var (
	ErrTokenExpired  = errors.New("token has expired")
	ErrTokenInvalid  = errors.New("token is invalid")
	ErrTokenMissing  = errors.New("authorization token is missing")
)

// getSecretKey reads the JWT secret from environment variables.
// All services must have the same JWT_SECRET env variable.
func getSecretKey() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "eticaret-super-secret-key-change-in-production"
	}
	return []byte(secret)
}

// GenerateToken creates a signed JWT token for the given user.
// Only AuthService should call this function.
func GenerateToken(userID int, email, role, firstName, lastName string) (string, error) {
	expirationHours := 24
	claims := &Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		FirstName: firstName,
		LastName:  lastName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expirationHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "eticaret-auth-service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSecretKey())
}

// ValidateToken parses and validates a JWT token string.
// All services (except AuthService for generation) call this to protect endpoints.
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return getSecretKey(), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
