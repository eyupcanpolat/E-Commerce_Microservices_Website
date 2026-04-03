// Package middleware provides network isolation for product-service.
// See auth-service/middleware for full documentation of the isolation model.
package middleware

import (
	"fmt"
	"net/http"
	"os"

	"eticaret/shared/logger"
	"eticaret/shared/response"
)

const InternalSecretHeader = "X-Internal-Secret"

func getInternalSecret() string {
	s := os.Getenv("INTERNAL_SECRET")
	if s == "" {
		s = "internal-gateway-secret-change-in-prod"
	}
	return s
}

// NetworkIsolation rejects requests that don't have the X-Internal-Secret header.
// This ensures only the API Gateway can call this service.
func NetworkIsolation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := r.Header.Get(InternalSecretHeader)
		if secret == "" || secret != getInternalSecret() {
			logger.Warn("Direct access attempt blocked",
				"service", "product-service",
				"path", r.URL.Path,
				"remote", r.RemoteAddr,
			)
			response.Forbidden(w, "Bu servise doğrudan erişim yasaktır")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserID reads the user ID from gateway-injected header.
func GetUserID(r *http.Request) int {
	idStr := r.Header.Get("X-User-ID")
	if idStr == "" {
		return 0
	}
	id := 0
	fmt.Sscanf(idStr, "%d", &id)
	return id
}

// GetUserRole reads the user role from gateway-injected header.
func GetUserRole(r *http.Request) string {
	return r.Header.Get("X-User-Role")
}

// GetUserEmail reads the user email from gateway-injected header.
func GetUserEmail(r *http.Request) string {
	return r.Header.Get("X-User-Email")
}

// RequireAdmin checks if the gateway confirmed admin role.
// Must be used AFTER NetworkIsolation (we trust gateway-set headers only).
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUserRole(r) != "admin" {
			response.Forbidden(w, "Bu işlem için admin yetkisi gerekiyor")
			return
		}
		next.ServeHTTP(w, r)
	})
}
