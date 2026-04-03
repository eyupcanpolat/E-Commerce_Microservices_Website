// Package repository — MongoDB implementation of AddressRepository.
package repository

import (
	"context"
	"errors"
	"time"

	"eticaret/address-service/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoAddressRepository struct {
	coll     *mongo.Collection
	counters *mongo.Collection
}

// NewMongoAddressRepository creates a MongoDB-backed AddressRepository.
func NewMongoAddressRepository(db *mongo.Database) AddressRepository {
	repo := &mongoAddressRepository{
		coll:     db.Collection("addresses"),
		counters: db.Collection("counters"),
	}
	repo.ensureIndexes()
	return repo
}

func (r *mongoAddressRepository) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
	})
}

func (r *mongoAddressRepository) nextID(ctx context.Context) (int, error) {
	var result struct {
		Seq int `bson:"seq"`
	}
	err := r.counters.FindOneAndUpdate(
		ctx,
		bson.M{"_id": "addresses"},
		bson.M{"$inc": bson.M{"seq": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&result)
	return result.Seq, err
}

func (r *mongoAddressRepository) GetByUserID(userID int) ([]model.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.coll.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var addresses []model.Address
	if err := cursor.All(ctx, &addresses); err != nil {
		return nil, err
	}
	if addresses == nil {
		addresses = []model.Address{}
	}
	return addresses, nil
}

func (r *mongoAddressRepository) GetByID(id int) (*model.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var addr model.Address
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&addr)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("address not found")
	}
	return &addr, err
}

func (r *mongoAddressRepository) Create(addr *model.Address) (*model.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, err := r.nextID(ctx)
	if err != nil {
		return nil, err
	}
	addr.ID = id
	addr.CreatedAt = time.Now()
	addr.UpdatedAt = time.Now()

	// If new address is default, unset existing defaults for this user
	if addr.IsDefault {
		r.coll.UpdateMany(ctx,
			bson.M{"user_id": addr.UserID},
			bson.M{"$set": bson.M{"is_default": false}},
		)
	}

	_, err = r.coll.InsertOne(ctx, addr)
	return addr, err
}

func (r *mongoAddressRepository) Update(id, userID int, req model.AddressRequest) (*model.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// If updating to default, unset other defaults for this user first
	if req.IsDefault {
		r.coll.UpdateMany(ctx,
			bson.M{"user_id": userID, "_id": bson.M{"$ne": id}},
			bson.M{"$set": bson.M{"is_default": false}},
		)
	}

	update := bson.M{
		"title":         req.Title,
		"first_name":    req.FirstName,
		"last_name":     req.LastName,
		"phone":         req.Phone,
		"address_line1": req.AddressLine1,
		"address_line2": req.AddressLine2,
		"city":          req.City,
		"state":         req.State,
		"postal_code":   req.PostalCode,
		"country":       req.Country,
		"is_default":    req.IsDefault,
		"updated_at":    time.Now(),
	}

	var addr model.Address
	err := r.coll.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": update},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&addr)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("address not found or access denied")
	}
	return &addr, err
}

func (r *mongoAddressRepository) Delete(id, userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.coll.DeleteOne(ctx, bson.M{"_id": id, "user_id": userID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("address not found or access denied")
	}
	return nil
}
