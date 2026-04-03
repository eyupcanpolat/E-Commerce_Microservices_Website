// main.go — API Gateway (Dispatcher)
//
// ARCHITECTURE — Network Isolation:
//
//   Internet ──► Gateway :8080 ──► internal Docker network ──► microservices
//                    │                                               │
//                    │ 1. JWT validated HERE (centrally)             │
//                    │ 2. X-Internal-Secret injected                 │
//                    │ 3. X-User-* headers set                       │
//                    ▼                                               ▼
//              Frontend talks             Microservices ONLY accept
//              ONLY to :8080              requests with X-Internal-Secret
//
// Route Auth Rules:
//   PUBLIC  → /auth/*, GET /products/*, GET /products/featured, GET /products/search
//   JWT     → /addresses/*, /orders/*, POST|DELETE /products/*
//   ADMIN   → DELETE /products/*, PUT /orders/*/status
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	gwMiddleware "eticaret/api-gateway/internal/middleware"
	"eticaret/api-gateway/internal/ratelimit"
	"eticaret/shared/logger"
)

func main() {
	// Backend service URLs — Docker internal hostnames in production,
	// localhost in local dev
	authURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8081")
	productURL := getEnv("PRODUCT_SERVICE_URL", "http://localhost:8082")
	addressURL := getEnv("ADDRESS_SERVICE_URL", "http://localhost:8083")
	orderURL := getEnv("ORDER_SERVICE_URL", "http://localhost:8084")

	mux := http.NewServeMux()

	// ── Gateway health endpoint ──────────────────────────────────────────────
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"service": "api-gateway",
			"status":  "ok",
			"isolation": map[string]string{
				"model":  "X-Internal-Secret header injection",
				"detail": "Microservices reject requests without the internal secret",
			},
			"routes": map[string]string{
				"/auth":      authURL,
				"/products":  productURL,
				"/addresses": addressURL,
				"/orders":    orderURL,
			},
		})
	})

	// ── Auth Service routes ─────────────────────────
	// Register, Login are public, Profile update requires JWT
	authProxy := routeAuth(authURL)
	mux.Handle("/auth", authProxy)
	mux.Handle("/auth/", authProxy)

	// ── Product Service routes ────────────────────────────────────────────────
	// GET routes: public (product listing, detail, search, featured)
	// POST/DELETE routes: require JWT + admin role
	productProxy := routeProducts(productURL)
	mux.Handle("/products", productProxy)
	mux.Handle("/products/", productProxy)

	// ── Address Service routes — ALL require JWT ──────────────────────────────
	addressProxy := gwMiddleware.JWTAuth(injectSecret(reverseProxy(addressURL)))
	mux.Handle("/addresses", addressProxy)
	mux.Handle("/addresses/", addressProxy)

	// ── Order Service routes — ALL require JWT ────────────────────────────────
	orderProxy := routeOrders(orderURL)
	mux.Handle("/orders", orderProxy)
	mux.Handle("/orders/", orderProxy)

	// ── Apply global middleware to everything ─────────────────────────────────────
	// Rate limiter: 60 req/min per IP (override with RATE_LIMIT_PER_MINUTE env)
	limiter := ratelimit.NewLimiterFromEnv()

	handler := gwMiddleware.Chain(
		mux,
		gwMiddleware.CORS,
		gwMiddleware.RequestLogger,
		limiter.Middleware,
	)

	port := getEnv("GATEWAY_PORT", "8080")
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("API Gateway starting",
		"port", port,
		"auth_url", authURL,
		"product_url", productURL,
		"address_url", addressURL,
		"order_url", orderURL,
		"isolation", "X-Internal-Secret enforced on all microservices",
	)

	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("API Gateway failed", "error", err)
	}
}

// routeAuth applies JWT only for PUT /auth/profile
func routeAuth(authURL string) http.Handler {
	proxy := reverseProxy(authURL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/profile") {
			gwMiddleware.JWTAuth(injectSecret(proxy)).ServeHTTP(w, r)
			return
		}
		// Public
		injectSecret(proxy).ServeHTTP(w, r)
	})
}

// routeProducts applies different auth rules based on method.
//   GET  /products/* → public
//   POST /products   → JWT + admin
//   DELETE /products/* → JWT + admin
func routeProducts(productURL string) http.Handler {
	proxy := reverseProxy(productURL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Public — no auth needed
			injectSecret(proxy).ServeHTTP(w, r)
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			// Admin only
			gwMiddleware.JWTAuth(
				gwMiddleware.RequireRole("admin")(
					injectSecret(proxy),
				),
			).ServeHTTP(w, r)
		default:
			injectSecret(proxy).ServeHTTP(w, r)
		}
	})
}

// routeOrders applies JWT to all order routes, and admin-only to PUT status.
func routeOrders(orderURL string) http.Handler {
	proxy := reverseProxy(orderURL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// PUT /orders/{id}/status → admin only
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/status") {
			gwMiddleware.JWTAuth(
				gwMiddleware.RequireRole("admin")(
					injectSecret(proxy),
				),
			).ServeHTTP(w, r)
			return
		}
		// All other order routes require JWT (any role)
		gwMiddleware.JWTAuth(injectSecret(proxy)).ServeHTTP(w, r)
	})
}

// injectSecret wraps a handler to add X-Internal-Secret before forwarding.
// This is the key mechanism for network isolation.
func injectSecret(next http.Handler) http.Handler {
	return gwMiddleware.InjectInternalSecret(next)
}

// reverseProxy creates a reverse proxy handler for the given target URL.
func reverseProxy(target string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		logger.Fatal("Invalid proxy target", "target", target)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("Proxy backend unavailable",
			"target", target,
			"path", r.URL.Path,
			"error", err,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Servis şu an kullanılamıyor",
		})
	}
	return proxy
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
