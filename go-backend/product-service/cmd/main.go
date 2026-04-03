// main.go — Product Service
// NETWORK ISOLATION: Only accepts requests with X-Internal-Secret from API Gateway.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"eticaret/product-service/internal/handler"
	"eticaret/product-service/internal/middleware"
	"eticaret/product-service/internal/repository"
	"eticaret/product-service/internal/service"
	"eticaret/shared/logger"
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

	db := client.Database("eticaret")
	productRepo := repository.NewMongoProductRepository(db)
	productSvc := service.NewProductService(productRepo)
	productHandler := handler.NewProductHandler(productSvc)

	mux := http.NewServeMux()

	// ALL routes require X-Internal-Secret (NetworkIsolation)
	mux.Handle("GET /health", middleware.NetworkIsolation(http.HandlerFunc(productHandler.Health)))
	mux.Handle("GET /products", middleware.NetworkIsolation(http.HandlerFunc(productHandler.ListProducts)))
	mux.Handle("GET /products/featured", middleware.NetworkIsolation(http.HandlerFunc(productHandler.GetFeatured)))
	mux.Handle("GET /products/search", middleware.NetworkIsolation(http.HandlerFunc(productHandler.Search)))
	mux.Handle("GET /products/{id}", middleware.NetworkIsolation(http.HandlerFunc(productHandler.GetProduct)))
	mux.Handle("GET /products/slug/{slug}", middleware.NetworkIsolation(http.HandlerFunc(productHandler.GetProductBySlug)))

	// Admin routes: Gateway already verified JWT + admin role
	adminCreate := middleware.NetworkIsolation(middleware.RequireAdmin(http.HandlerFunc(productHandler.CreateProduct)))
	adminUpdate := middleware.NetworkIsolation(middleware.RequireAdmin(http.HandlerFunc(productHandler.UpdateProduct)))
	adminDelete := middleware.NetworkIsolation(middleware.RequireAdmin(http.HandlerFunc(productHandler.DeleteProduct)))
	mux.Handle("POST /products", adminCreate)
	mux.Handle("PUT /products/{id}", adminUpdate)
	mux.Handle("DELETE /products/{id}", adminDelete)

	port := getEnv("PRODUCT_SERVICE_PORT", "8082")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Product Service başlatılıyor",
		"port", port,
		"network_isolation", "X-Internal-Secret required",
		"jwt_validation", "gateway-only",
		"db", "MongoDB/eticaret",
	)
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Product Service başlatılamadı", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
