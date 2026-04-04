// main.go — Auth Service
// NETWORK ISOLATION: This service only accepts requests with X-Internal-Secret header.
// It does NOT validate JWT — that's the gateway's job.
// It reads user identity from X-User-* headers set by the gateway.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"eticaret/auth-service/internal/handler"
	"eticaret/auth-service/internal/middleware"
	"eticaret/auth-service/internal/repository"
	"eticaret/auth-service/internal/service"
	"eticaret/shared/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		logger.Fatal("MongoDB bağlantısı kurulamadı", "error", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		logger.Fatal("MongoDB ping başarısız", "uri", mongoURI, "error", err)
	}
	logger.Info("MongoDB bağlantısı kuruldu", "uri", mongoURI)

	db := client.Database("eticaret_auth")
	userRepo := repository.NewMongoUserRepository(db)
	authSvc := service.NewAuthService(userRepo)
	authHandler := handler.NewAuthHandler(authSvc)

	mux := http.NewServeMux()

	// Prometheus metrics — açık endpoint (Prometheus Docker ağından erişir)
	mux.Handle("GET /metrics", promhttp.Handler())

	// ALL routes are wrapped with NetworkIsolation — only gateway can call us
	mux.Handle("GET /health", middleware.NetworkIsolation(http.HandlerFunc(authHandler.Health)))
	mux.Handle("POST /auth/register", middleware.NetworkIsolation(http.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /auth/login", middleware.NetworkIsolation(http.HandlerFunc(authHandler.Login)))
	mux.Handle("PUT /auth/profile", middleware.NetworkIsolation(http.HandlerFunc(authHandler.UpdateProfile)))

	port := getEnv("AUTH_SERVICE_PORT", "8081")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Auth Service başlatılıyor",
		"port", port,
		"network_isolation", "X-Internal-Secret required on all routes",
		"jwt_validation", "gateway-only",
		"db", "MongoDB/eticaret",
	)

	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Auth Service başlatılamadı", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
