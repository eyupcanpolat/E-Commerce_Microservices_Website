// main.go — Order Service with network isolation
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"eticaret/order-service/internal/handler"
	"eticaret/order-service/internal/middleware"
	"eticaret/order-service/internal/repository"
	"eticaret/order-service/internal/service"
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

	productSvcURL := getEnv("PRODUCT_SERVICE_URL", "http://localhost:8082")

	db := client.Database("eticaret_orders")
	orderRepo := repository.NewMongoOrderRepository(db)
	orderSvc := service.NewOrderService(orderRepo, productSvcURL)
	orderHandler := handler.NewOrderHandler(orderSvc)

	mux := http.NewServeMux()

	// Prometheus metrics — açık endpoint
	mux.Handle("GET /metrics", promhttp.Handler())

	mux.Handle("GET /health", middleware.NetworkIsolation(http.HandlerFunc(orderHandler.Health)))

	protect := func(h http.HandlerFunc) http.Handler {
		return middleware.NetworkIsolation(middleware.RequireUser(h))
	}
	protectAdmin := func(h http.HandlerFunc) http.Handler {
		return middleware.NetworkIsolation(middleware.RequireUser(middleware.RequireAdmin(h)))
	}

	mux.Handle("GET /orders", protect(orderHandler.GetOrders))
	mux.Handle("POST /orders", protect(orderHandler.CreateOrder))
	mux.Handle("GET /orders/{id}", protect(orderHandler.GetOrder))
	mux.Handle("GET /orders/number/{orderNumber}", protect(orderHandler.GetOrderByNumber))
	mux.Handle("POST /orders/{orderNumber}/cancel", protect(orderHandler.CancelOrder))
	mux.Handle("PUT /orders/{id}/status", protectAdmin(orderHandler.UpdateStatus))

	port := getEnv("ORDER_SERVICE_PORT", "8084")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Order Service başlatılıyor",
		"port", port,
		"network_isolation", "X-Internal-Secret required",
		"jwt_validation", "gateway-only",
		"product_service_url", productSvcURL,
		"db", "MongoDB/eticaret",
	)
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Order Service başlatılamadı", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
