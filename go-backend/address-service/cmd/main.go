// main.go — Address Service with network isolation
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"eticaret/address-service/internal/handler"
	"eticaret/address-service/internal/middleware"
	"eticaret/address-service/internal/repository"
	"eticaret/address-service/internal/service"
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

	db := client.Database("eticaret_addresses")
	addrRepo := repository.NewMongoAddressRepository(db)
	addrSvc := service.NewAddressService(addrRepo)
	addrHandler := handler.NewAddressHandler(addrSvc)

	mux := http.NewServeMux()

	// Prometheus metrics — açık endpoint
	mux.Handle("GET /metrics", promhttp.Handler())

	mux.Handle("GET /health", middleware.NetworkIsolation(http.HandlerFunc(addrHandler.Health)))

	protect := func(h http.HandlerFunc) http.Handler {
		return middleware.NetworkIsolation(middleware.RequireUser(h))
	}

	mux.Handle("GET /addresses", protect(addrHandler.GetAddresses))
	mux.Handle("POST /addresses", protect(addrHandler.CreateAddress))
	mux.Handle("GET /addresses/{id}", protect(addrHandler.GetAddress))
	mux.Handle("PUT /addresses/{id}", protect(addrHandler.UpdateAddress))
	mux.Handle("DELETE /addresses/{id}", protect(addrHandler.DeleteAddress))

	port := getEnv("ADDRESS_SERVICE_PORT", "8083")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Address Service başlatılıyor",
		"port", port,
		"network_isolation", "X-Internal-Secret required",
		"jwt_validation", "gateway-only",
		"db", "MongoDB/eticaret",
	)
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Address Service başlatılamadı", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
