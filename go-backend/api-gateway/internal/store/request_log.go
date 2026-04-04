// Package store provides the gateway's own isolated MongoDB collection
// for storing request logs and access records.
//
// ISOLATION: Gateway uses "eticaret_gateway" database — completely separate
// from microservice databases (eticaret_auth, eticaret_products, etc.)
package store

import (
	"context"
	"os"
	"time"

	"eticaret/shared/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RequestLog is a single gateway access log entry stored in MongoDB.
type RequestLog struct {
	ID         interface{} `bson:"_id,omitempty"`
	Method     string      `bson:"method"`
	Path       string      `bson:"path"`
	StatusCode int         `bson:"status_code"`
	DurationMs int64       `bson:"duration_ms"`
	IP         string      `bson:"ip"`
	UserID     string      `bson:"user_id,omitempty"`
	UserRole   string      `bson:"user_role,omitempty"`
	Service    string      `bson:"service"` // which microservice was proxied to
	Error      string      `bson:"error,omitempty"`
	Timestamp  time.Time   `bson:"timestamp"`
}

// GatewayStore handles gateway's own isolated database operations.
type GatewayStore struct {
	collection *mongo.Collection
}

// NewGatewayStore connects to MongoDB and returns the gateway store.
// Uses "eticaret_gateway" DB — isolated from all microservice DBs.
func NewGatewayStore() *GatewayStore {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		logger.Error("Gateway MongoDB connect failed", "error", err)
		return &GatewayStore{} // graceful degradation — gateway still works without DB
	}

	if err := client.Ping(ctx, nil); err != nil {
		logger.Error("Gateway MongoDB ping failed", "error", err)
		return &GatewayStore{}
	}

	// Gateway uses its OWN isolated database
	db := client.Database("eticaret_gateway")
	col := db.Collection("request_logs")

	logger.Info("Gateway store connected", "database", "eticaret_gateway", "collection", "request_logs")
	return &GatewayStore{collection: col}
}

// LogRequest persists a request log entry to MongoDB.
// Non-blocking — runs in background goroutine.
func (s *GatewayStore) LogRequest(log RequestLog) {
	if s.collection == nil {
		return // graceful degradation
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		log.Timestamp = time.Now()
		if _, err := s.collection.InsertOne(ctx, log); err != nil {
			logger.Error("Gateway log insert failed", "error", err)
		}
	}()
}

// GetRecentLogs returns the last N request logs.
func (s *GatewayStore) GetRecentLogs(limit int64) ([]RequestLog, error) {
	if s.collection == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().
		SetSort(map[string]int{"timestamp": -1}).
		SetLimit(limit)

	cursor, err := s.collection.Find(ctx, map[string]interface{}{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []RequestLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}
