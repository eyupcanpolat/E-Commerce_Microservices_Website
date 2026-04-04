// Package middleware provides gateway-level middleware for the API Gateway.
//
// Architecture:
//   - ALL JWT validation happens HERE (in the gateway), never in microservices
//   - Gateway injects X-Internal-Secret header to every forwarded request
//   - Microservices reject any request missing this secret (network isolation)
//   - External world can only reach port 8080 (gateway port)
package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	sharedJWT "eticaret/shared/jwt"
	"eticaret/shared/logger"
	"eticaret/shared/response"
)

// RequestLogger arayüzü — store bağımlılığını tersine çevirir (DIP).
// store paketi yerine bu arayüzü kullanırız, döngüsel import olmaz.
type RequestStorer interface {
	LogRequest(log RequestLog)
}

// RequestLog middleware için minimal log yapısı.
type RequestLog struct {
	Method     string
	Path       string
	StatusCode int
	DurationMs int64
	IP         string
	UserID     string
	UserRole   string
}

// getInternalSecret reads the shared internal secret from environment.
// Gateway injects this on every forwarded request.
// Microservices verify it — rejecting anything without it.
func GetInternalSecret() string {
	s := os.Getenv("INTERNAL_SECRET")
	if s == "" {
		s = "internal-gateway-secret-change-in-prod"
	}
	return s
}

// InternalSecretHeader is the header name used for network isolation.
const InternalSecretHeader = "X-Internal-Secret"

// RequestLogger logs every incoming request at the gateway level.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("→ Gateway request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
		)
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestLoggerWithStore logs every request and persists it via the storer.
// Use this instead of RequestLogger when you want DB-backed access logs.
func RequestLoggerWithStore(storer RequestStorer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			logger.Info("→ Gateway request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote", r.RemoteAddr,
			)

			next.ServeHTTP(rw, r)

			storer.LogRequest(RequestLog{
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: rw.statusCode,
				DurationMs: time.Since(start).Milliseconds(),
				IP:         r.RemoteAddr,
				UserID:     r.Header.Get("X-User-ID"),
				UserRole:   r.Header.Get("X-User-Role"),
			})
		})
	}
}

// InjectInternalSecret adds the X-Internal-Secret header to every request
// before it is forwarded to a microservice. This allows microservices to
// verify the request came through the gateway (network isolation).
func InjectInternalSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set(InternalSecretHeader, GetInternalSecret())
		next.ServeHTTP(w, r)
	})
}

// JWTAuth validates the Bearer token and injects user identity headers.
// This is the SINGLE place where JWT is validated in the whole system.
// On success, sets:
//   - X-User-ID:    e.g. "2"
//   - X-User-Email: e.g. "ahmet@example.com"
//   - X-User-Role:  e.g. "customer" or "admin"
//
// Microservices read these headers instead of re-validating JWT.
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Unauthorized(w, "Authorization header eksik")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			response.Unauthorized(w, "Authorization formatı geçersiz. Beklenen: Bearer <token>")
			return
		}

		claims, err := sharedJWT.ValidateToken(parts[1])
		if err != nil {
			switch err {
			case sharedJWT.ErrTokenExpired:
				response.Unauthorized(w, "Token süresi dolmuş. Lütfen tekrar giriş yapın.")
			default:
				response.Unauthorized(w, "Geçersiz token")
			}
			return
		}

		// Inject parsed user identity into request headers for microservices
		// Microservices trust these headers because they are sent over internal network
		r.Header.Set("X-User-ID", fmt.Sprintf("%d", claims.UserID))
		r.Header.Set("X-User-Email", claims.Email)
		r.Header.Set("X-User-Role", claims.Role)
		r.Header.Set("X-User-FirstName", claims.FirstName)
		r.Header.Set("X-User-LastName", claims.LastName)

		logger.Info("JWT validated at gateway",
			"user_id", claims.UserID,
			"role", claims.Role,
			"path", r.URL.Path,
		)

		next.ServeHTTP(w, r)
	})
}

// RequireRole ensures only users with the specified role can pass.
// Must be used AFTER JWTAuth middleware.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := r.Header.Get("X-User-Role")
			if userRole != role {
				response.Forbidden(w, "Bu işlem için yetkiniz yok")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS adds cross-origin headers to all gateway responses.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Chain applies multiple middleware in order (first applied = outermost).
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
