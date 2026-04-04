// Package repository — MongoDB implementation of ProductRepository.
package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"eticaret/product-service/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoProductRepository struct {
	coll     *mongo.Collection
	counters *mongo.Collection
}

// NewMongoProductRepository creates a MongoDB-backed ProductRepository.
func NewMongoProductRepository(db *mongo.Database) ProductRepository {
	repo := &mongoProductRepository{
		coll:     db.Collection("products"),
		counters: db.Collection("counters"),
	}
	repo.ensureIndexes()
	return repo
}

func (r *mongoProductRepository) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "is_active", Value: 1}}},
		{Keys: bson.D{{Key: "is_featured", Value: 1}}},
		{Keys: bson.D{{Key: "price", Value: 1}}},
		{Keys: bson.D{{Key: "category_id", Value: 1}, {Key: "is_active", Value: 1}}},
		{Keys: bson.D{{Key: "name", Value: "text"}, {Key: "description", Value: "text"}}},
	})
}

func (r *mongoProductRepository) nextID(ctx context.Context) (int, error) {
	var result struct {
		Seq int `bson:"seq"`
	}
	err := r.counters.FindOneAndUpdate(
		ctx,
		bson.M{"_id": "products"},
		bson.M{"$inc": bson.M{"seq": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&result)
	return result.Seq, err
}

func (r *mongoProductRepository) GetAll(filter model.ProductFilter, page, perPage int) (*model.PaginatedProducts, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := bson.M{"is_active": true}

	if filter.CategoryID != nil {
		query["category_id"] = *filter.CategoryID
	}
	if filter.MinPrice != nil {
		query["price"] = bson.M{"$gte": *filter.MinPrice}
	}
	if filter.MaxPrice != nil {
		if existing, ok := query["price"]; ok {
			query["price"] = bson.M{"$gte": existing.(bson.M)["$gte"], "$lte": *filter.MaxPrice}
		} else {
			query["price"] = bson.M{"$lte": *filter.MaxPrice}
		}
	}
	if filter.InStock {
		query["stock_status"] = bson.M{"$ne": "out_of_stock"}
	}
	if filter.IsFeatured != nil {
		query["is_featured"] = *filter.IsFeatured
	}
	if filter.Search != "" {
		query["$or"] = bson.A{
			bson.M{"name": primitive.Regex{Pattern: filter.Search, Options: "i"}},
			bson.M{"description": primitive.Regex{Pattern: filter.Search, Options: "i"}},
		}
	}

	// Sort
	sortDoc := bson.D{{Key: "created_at", Value: -1}}
	switch strings.ToLower(filter.Sort) {
	case "price_asc":
		sortDoc = bson.D{{Key: "price", Value: 1}}
	case "price_desc":
		sortDoc = bson.D{{Key: "price", Value: -1}}
	case "popular":
		sortDoc = bson.D{{Key: "view_count", Value: -1}}
	}

	total, err := r.coll.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	skip := int64((page - 1) * perPage)

	cursor, err := r.coll.Find(ctx, query,
		options.Find().SetSort(sortDoc).SetSkip(skip).SetLimit(int64(perPage)),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []model.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}
	if products == nil {
		products = []model.Product{}
	}

	totalInt := int(total)
	totalPages := (totalInt + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	return &model.PaginatedProducts{
		Data:       products,
		Total:      totalInt,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func (r *mongoProductRepository) GetByID(id int) (*model.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var p model.Product
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("product not found")
	}
	return &p, err
}

func (r *mongoProductRepository) GetBySlug(slug string) (*model.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var p model.Product
	err := r.coll.FindOne(ctx, bson.M{"slug": slug}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("product not found")
	}
	return &p, err
}

func (r *mongoProductRepository) GetFeatured(limit int) ([]model.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.coll.Find(ctx,
		bson.M{"is_featured": true, "is_active": true},
		options.Find().SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []model.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}
	if products == nil {
		products = []model.Product{}
	}
	return products, nil
}

func (r *mongoProductRepository) Search(query string) ([]model.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"is_active": true,
		"$or": bson.A{
			bson.M{"name": primitive.Regex{Pattern: query, Options: "i"}},
			bson.M{"description": primitive.Regex{Pattern: query, Options: "i"}},
		},
	}

	cursor, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []model.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}
	if products == nil {
		products = []model.Product{}
	}
	return products, nil
}

func (r *mongoProductRepository) Create(p *model.Product) (*model.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, err := r.nextID(ctx)
	if err != nil {
		return nil, err
	}
	p.ID = id
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	p.IsActive = true

	_, err = r.coll.InsertOne(ctx, p)
	return p, err
}

func (r *mongoProductRepository) Update(id int, updated *model.Product) (*model.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Preserve fields that shouldn't be overwritten
	existing, err := r.GetByID(id)
	if err != nil {
		return nil, errors.New("product not found")
	}
	updated.ID = id
	updated.CreatedAt = existing.CreatedAt
	updated.UpdatedAt = time.Now()
	updated.IsActive = existing.IsActive
	updated.ViewCount = existing.ViewCount

	_, err = r.coll.ReplaceOne(ctx, bson.M{"_id": id}, updated)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *mongoProductRepository) Delete(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("product not found")
	}
	return nil
}
