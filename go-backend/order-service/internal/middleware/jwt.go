// Package middleware provides network isolation for order-service.
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

// NetworkIsolation rejects requests without X-Internal-Secret header.
func NetworkIsolation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := r.Header.Get(InternalSecretHeader)
		if secret == "" || secret != getInternalSecret() {
			logger.Warn("Direct access attempt blocked",
				"service", "order-service",
				"path", r.URL.Path,
				"remote", r.RemoteAddr,
			)
			response.Forbidden(w, "Bu servise doğrudan erişim yasaktır")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserID reads user ID from gateway-injected header.
func GetUserID(r *http.Request) int {
	idStr := r.Header.Get("X-User-ID")
	if idStr == "" {
		return 0
	}
	id := 0
	fmt.Sscanf(idStr, "%d", &id)
	return id
}

// GetUserRole reads user role from gateway-injected header.
func GetUserRole(r *http.Request) string {
	return r.Header.Get("X-User-Role")
}

// RequireUser ensures request has an authenticated user.
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUserID(r) == 0 {
			response.Unauthorized(w, "Bu işlem için giriş yapmanız gerekiyor")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin ensures user has admin role (set by gateway).
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUserRole(r) != "admin" {
			response.Forbidden(w, "Bu işlem için admin yetkisi gerekiyor")
			return
		}
		next.ServeHTTP(w, r)
	})
}
