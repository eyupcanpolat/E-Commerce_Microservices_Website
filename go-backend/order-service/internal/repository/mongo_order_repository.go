// Package repository — MongoDB implementation of OrderRepository.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"eticaret/order-service/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoOrderRepository struct {
	coll     *mongo.Collection
	counters *mongo.Collection
}

// NewMongoOrderRepository creates a MongoDB-backed OrderRepository.
func NewMongoOrderRepository(db *mongo.Database) OrderRepository {
	repo := &mongoOrderRepository{
		coll:     db.Collection("orders"),
		counters: db.Collection("counters"),
	}
	repo.ensureIndexes()
	return repo
}

func (r *mongoOrderRepository) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "order_number", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
}

func (r *mongoOrderRepository) nextID(ctx context.Context) (int, error) {
	var result struct {
		Seq int `bson:"seq"`
	}
	err := r.counters.FindOneAndUpdate(
		ctx,
		bson.M{"_id": "orders"},
		bson.M{"$inc": bson.M{"seq": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&result)
	return result.Seq, err
}

func (r *mongoOrderRepository) GetByUserID(userID int) ([]model.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.coll.Find(ctx,
		bson.M{"user_id": userID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []model.Order
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}
	if orders == nil {
		orders = []model.Order{}
	}
	return orders, nil
}

func (r *mongoOrderRepository) GetByID(id int) (*model.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var order model.Order
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&order)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("order not found")
	}
	return &order, err
}

func (r *mongoOrderRepository) GetByOrderNumber(orderNumber string) (*model.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var order model.Order
	err := r.coll.FindOne(ctx, bson.M{"order_number": orderNumber}).Decode(&order)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("order not found")
	}
	return &order, err
}

func (r *mongoOrderRepository) Create(order *model.Order) (*model.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, err := r.nextID(ctx)
	if err != nil {
		return nil, err
	}
	order.ID = id
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	order.Status = model.StatusPending
	order.PaymentStatus = "pending"
	order.OrderNumber = fmt.Sprintf("ORD-%d-%04d", time.Now().Year(), id)

	for i := range order.Items {
		order.Items[i].ID = i + 1
		order.Items[i].OrderID = id
	}

	_, err = r.coll.InsertOne(ctx, order)
	return order, err
}

func (r *mongoOrderRepository) UpdateStatus(id int, status model.OrderStatus) (*model.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	update := bson.M{
		"status":     status,
		"updated_at": now,
	}
	if status == model.StatusShipped {
		update["shipped_at"] = now
	}
	if status == model.StatusDelivered {
		update["delivered_at"] = now
	}

	var order model.Order
	err := r.coll.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": update},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&order)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("order not found")
	}
	return &order, err
}

func (r *mongoOrderRepository) Cancel(id, userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	order, err := r.GetByID(id)
	if err != nil || order.UserID != userID {
		return errors.New("sipariş bulunamadı veya erişim reddedildi")
	}
	if order.Status != model.StatusPending && order.Status != model.StatusProcessing {
		return errors.New("bu aşamadaki sipariş iptal edilemez")
	}

	_, err = r.coll.UpdateOne(ctx,
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": bson.M{"status": model.StatusCancelled, "updated_at": time.Now()}},
	)
	return err
}
