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
	"strconv"
	"time"

	"eticaret/api-gateway/internal/handler"
	gwMiddleware "eticaret/api-gateway/internal/middleware"
	"eticaret/api-gateway/internal/ratelimit"
	"eticaret/api-gateway/internal/store"
	"eticaret/shared/logger"
)

func main() {
	// Backend service URLs — Docker internal hostnames in production,
	// localhost in local dev
	authURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8081")
	productURL := getEnv("PRODUCT_SERVICE_URL", "http://localhost:8082")
	addressURL := getEnv("ADDRESS_SERVICE_URL", "http://localhost:8083")
	orderURL := getEnv("ORDER_SERVICE_URL", "http://localhost:8084")

	serviceURLs := map[string]string{
		"/auth":      authURL,
		"/products":  productURL,
		"/addresses": addressURL,
		"/orders":    orderURL,
	}

	// Gateway'in kendi izole MongoDB store'u (eticaret_gateway DB)
	gatewayStore := store.NewGatewayStore()

	mux := http.NewServeMux()

	// ── Gateway health endpoint ──────────────────────────────────────────────
	mux.HandleFunc("GET /health", handler.HealthHandler(serviceURLs))

	// ── Gateway request logs — izole DB'den son logları döner ───────────────
	mux.HandleFunc("GET /gateway/logs", func(w http.ResponseWriter, r *http.Request) {
		limit := int64(100)
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.ParseInt(l, 10, 64); err == nil && n > 0 {
				limit = n
			}
		}
		logs, err := gatewayStore.GetRecentLogs(limit)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": logs})
	})

	// ── Auth Service routes ──────────────────────────────────────────────────
	// Register, Login are public; Profile update requires JWT
	authProxy := handler.NewAuthHandler(reverseProxy(authURL))
	mux.Handle("/auth", authProxy)
	mux.Handle("/auth/", authProxy)

	// ── Product Service routes ───────────────────────────────────────────────
	// GET routes: public | POST/PUT/DELETE: require JWT + admin role
	productProxy := handler.NewProductHandler(reverseProxy(productURL))
	mux.Handle("/products", productProxy)
	mux.Handle("/products/", productProxy)

	// ── Address Service routes — ALL require JWT ─────────────────────────────
	addressProxy := gwMiddleware.JWTAuth(gwMiddleware.InjectInternalSecret(reverseProxy(addressURL)))
	mux.Handle("/addresses", addressProxy)
	mux.Handle("/addresses/", addressProxy)

	// ── Order Service routes — ALL require JWT ───────────────────────────────
	orderProxy := handler.NewOrderHandler(reverseProxy(orderURL))
	mux.Handle("/orders", orderProxy)
	mux.Handle("/orders/", orderProxy)

	// ── Apply global middleware ──────────────────────────────────────────────
	limiter := ratelimit.NewLimiterFromEnv()
	h := gwMiddleware.Chain(
		mux,
		gwMiddleware.CORS,
		gwMiddleware.RequestLogger,
		limiter.Middleware,
	)

	port := getEnv("GATEWAY_PORT", "8080")
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      h,
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
