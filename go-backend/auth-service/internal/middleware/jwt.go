// Package middleware provides network isolation middleware for auth-service.
//
// SECURITY MODEL:
//   - auth-service MUST NOT be accessible directly from the internet
//   - All requests MUST come through the API Gateway
//   - Gateway injects X-Internal-Secret header on every forwarded request
//   - This middleware rejects any request that doesn't have the correct secret
//
// This means:
//   - Even if someone discovers port 8081, they CANNOT call auth-service
//   - Only the gateway (which is on the same Docker internal network) can call this service
//   - JWT is validated at the gateway level; this service trusts X-User-* headers
package middleware

import (
	"fmt"
	"net/http"
	"os"

	"eticaret/shared/logger"
	"eticaret/shared/response"
)

const InternalSecretHeader = "X-Internal-Secret"

// getInternalSecret reads the shared secret from environment.
// Must match the value configured in the API Gateway.
func getInternalSecret() string {
	s := os.Getenv("INTERNAL_SECRET")
	if s == "" {
		s = "internal-gateway-secret-change-in-prod"
	}
	return s
}

// NetworkIsolation is the primary security middleware for microservices.
// It ensures this service can ONLY be called by the API Gateway.
//
// How it works:
//  1. API Gateway sets X-Internal-Secret header before forwarding requests
//  2. This middleware checks the header value
//  3. If missing or wrong → 403 Forbidden (not 401, because this isn't auth related)
//  4. If correct → request is allowed to proceed
//
// This replaces per-service JWT validation. JWT is now ONLY validated at the gateway.
func NetworkIsolation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := r.Header.Get(InternalSecretHeader)
		if secret == "" || secret != getInternalSecret() {
			logger.Warn("Direct access attempt blocked",
				"path", r.URL.Path,
				"remote", r.RemoteAddr,
				"has_secret", secret != "",
			)
			// Return 403 with no details — don't reveal internal architecture
			response.Forbidden(w, "Bu servise doğrudan erişim yasaktır")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserID reads the user ID from the X-User-ID header set by the gateway.
// Returns 0 if not set (unauthenticated/public request passed through gateway).
func GetUserID(r *http.Request) int {
	idStr := r.Header.Get("X-User-ID")
	if idStr == "" {
		return 0
	}
	id := 0
	fmt.Sscanf(idStr, "%d", &id)
	return id
}

// GetUserRole reads the user role from the X-User-Role header set by the gateway.
func GetUserRole(r *http.Request) string {
	return r.Header.Get("X-User-Role")
}

// GetUserEmail reads the user email from the X-User-Email header set by the gateway.
func GetUserEmail(r *http.Request) string {
	return r.Header.Get("X-User-Email")
}

// RequireUser ensures the request has an authenticated user (set by gateway JWT validation).
// If X-User-ID is missing or 0, the request was not authenticated at the gateway.
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUserID(r) == 0 {
			response.Unauthorized(w, "Bu işlem için giriş yapmanız gerekiyor")
			return
		}
		next.ServeHTTP(w, r)
	})
}
