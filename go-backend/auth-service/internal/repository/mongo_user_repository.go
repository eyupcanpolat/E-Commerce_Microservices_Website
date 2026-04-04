// Package repository — MongoDB implementation of UserRepository.
package repository

import (
	"context"
	"errors"
	"time"

	"eticaret/auth-service/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoUserRepository struct {
	coll     *mongo.Collection
	counters *mongo.Collection
}

func NewMongoUserRepository(db *mongo.Database) UserRepository {
	repo := &mongoUserRepository{
		coll:     db.Collection("users"),
		counters: db.Collection("counters"),
	}
	repo.ensureIndexes()
	return repo
}

func (r *mongoUserRepository) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "is_active", Value: 1}}},
	})
}

// nextID atomically increments and returns the next integer ID for users.
func (r *mongoUserRepository) nextID(ctx context.Context) (int, error) {
	var result struct {
		Seq int `bson:"seq"`
	}
	err := r.counters.FindOneAndUpdate(
		ctx,
		bson.M{"_id": "users"},
		bson.M{"$inc": bson.M{"seq": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&result)
	return result.Seq, err
}

func (r *mongoUserRepository) FindByEmail(email string) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user model.User
	err := r.coll.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("user not found")
	}
	return &user, err
}

func (r *mongoUserRepository) FindByID(id int) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user model.User
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("user not found")
	}
	return &user, err
}

func (r *mongoUserRepository) EmailExists(email string) bool {
	_, err := r.FindByEmail(email)
	return err == nil
}

func (r *mongoUserRepository) Create(user *model.User) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, err := r.nextID(ctx)
	if err != nil {
		return nil, err
	}
	user.ID = id
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.Role == "" {
		user.Role = "customer"
	}
	user.IsActive = true

	_, err = r.coll.InsertOne(ctx, user)
	return user, err
}

func (r *mongoUserRepository) Update(id int, data map[string]interface{}) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fields := bson.M{"updated_at": time.Now()}
	if v, ok := data["first_name"]; ok {
		fields["first_name"] = v
	}
	if v, ok := data["last_name"]; ok {
		fields["last_name"] = v
	}
	if v, ok := data["phone"]; ok {
		fields["phone"] = v
	}
	if v, ok := data["password"]; ok {
		fields["password"] = v
	}

	var user model.User
	err := r.coll.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": fields},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("user not found")
	}
	return &user, err
}
