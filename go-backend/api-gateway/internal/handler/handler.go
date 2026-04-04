// Package handler provides the HTTP handler functions for the API Gateway.
//
// Handler'lar cmd/main.go'dan ayrılmış olup doğrudan test edilebilir.
// Her route grubu kendi handler constructor'ına sahiptir.
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"eticaret/api-gateway/internal/middleware"
)

// HealthHandler gateway'in sağlık durumunu döner.
// Bağımlılığı olmayan saf bir handler — kolayca test edilebilir.
func HealthHandler(serviceURLs map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"service": "api-gateway",
			"status":  "ok",
			"isolation": map[string]string{
				"model":  "X-Internal-Secret header injection",
				"detail": "Microservices reject requests without the internal secret",
			},
			"routes": serviceURLs,
		})
	}
}

// NewAuthHandler auth servisine yönlendiren handler'ı döner.
// Kurallar:
//   - PUT /auth/profile → JWT gerekli
//   - Diğer /auth/* → public
func NewAuthHandler(proxy http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/profile") {
			middleware.JWTAuth(middleware.InjectInternalSecret(proxy)).ServeHTTP(w, r)
			return
		}
		middleware.InjectInternalSecret(proxy).ServeHTTP(w, r)
	})
}

// NewProductHandler ürün servisine yönlendiren handler'ı döner.
// Kurallar:
//   - GET /products/* → public
//   - POST/PUT/DELETE /products/* → JWT + admin
func NewProductHandler(proxy http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			middleware.InjectInternalSecret(proxy).ServeHTTP(w, r)
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			middleware.JWTAuth(
				middleware.RequireRole("admin")(
					middleware.InjectInternalSecret(proxy),
				),
			).ServeHTTP(w, r)
		default:
			middleware.InjectInternalSecret(proxy).ServeHTTP(w, r)
		}
	})
}

// NewOrderHandler sipariş servisine yönlendiren handler'ı döner.
// Kurallar:
//   - PUT /orders/{id}/status → JWT + admin
//   - Diğer /orders/* → JWT (herhangi bir rol)
func NewOrderHandler(proxy http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/status") {
			middleware.JWTAuth(
				middleware.RequireRole("admin")(
					middleware.InjectInternalSecret(proxy),
				),
			).ServeHTTP(w, r)
			return
		}
		middleware.JWTAuth(middleware.InjectInternalSecret(proxy)).ServeHTTP(w, r)
	})
}
