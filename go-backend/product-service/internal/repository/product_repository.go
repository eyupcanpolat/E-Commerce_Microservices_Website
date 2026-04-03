// Package repository handles data access for ProductService.
package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"eticaret/product-service/internal/model"
)

// ProductRepository defines the interface for product data operations.
type ProductRepository interface {
	GetAll(filter model.ProductFilter, page, perPage int) (*model.PaginatedProducts, error)
	GetByID(id int) (*model.Product, error)
	GetBySlug(slug string) (*model.Product, error)
	GetFeatured(limit int) ([]model.Product, error)
	Search(query string) ([]model.Product, error)
	Create(p *model.Product) (*model.Product, error)
	Update(id int, p *model.Product) (*model.Product, error)
	Delete(id int) error
}

type jsonProductRepository struct {
	filePath string
	mu       sync.RWMutex
}

func NewProductRepository(filePath string) ProductRepository {
	return &jsonProductRepository{filePath: filePath}
}

func (r *jsonProductRepository) load() ([]model.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read products file: %w", err)
	}
	var products []model.Product
	return products, json.Unmarshal(data, &products)
}

func (r *jsonProductRepository) save(products []model.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0644)
}

func (r *jsonProductRepository) GetAll(filter model.ProductFilter, page, perPage int) (*model.PaginatedProducts, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}

	// Apply filters
	var filtered []model.Product
	for _, p := range all {
		if !p.IsActive {
			continue
		}
		if filter.CategoryID != nil && (p.CategoryID == nil || *p.CategoryID != *filter.CategoryID) {
			continue
		}
		if filter.MinPrice != nil && p.Price < *filter.MinPrice {
			continue
		}
		if filter.MaxPrice != nil && p.Price > *filter.MaxPrice {
			continue
		}
		if filter.InStock && p.StockStatus == "out_of_stock" {
			continue
		}
		if filter.IsFeatured != nil && p.IsFeatured != *filter.IsFeatured {
			continue
		}
		if filter.Search != "" {
			// Case-insensitive search in name and description
			if !containsIgnoreCase(p.Name, filter.Search) &&
				!containsIgnoreCase(p.Description, filter.Search) {
				continue
			}
		}
		filtered = append(filtered, p)
	}

	total := len(filtered)
	if page < 1 {
		page = 1
	}
	start := (page - 1) * perPage
	end := start + perPage
	if start >= total {
		start = total
	}
	if end > total {
		end = total
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	return &model.PaginatedProducts{
		Data:       filtered[start:end],
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func (r *jsonProductRepository) GetByID(id int) (*model.Product, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, errors.New("product not found")
}

func (r *jsonProductRepository) GetBySlug(slug string) (*model.Product, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].Slug == slug {
			return &all[i], nil
		}
	}
	return nil, errors.New("product not found")
}

func (r *jsonProductRepository) GetFeatured(limit int) ([]model.Product, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	var featured []model.Product
	for _, p := range all {
		if p.IsFeatured && p.IsActive {
			featured = append(featured, p)
			if len(featured) >= limit {
				break
			}
		}
	}
	return featured, nil
}

func (r *jsonProductRepository) Search(query string) ([]model.Product, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	var results []model.Product
	for _, p := range all {
		if p.IsActive && (containsIgnoreCase(p.Name, query) || containsIgnoreCase(p.Description, query)) {
			results = append(results, p)
		}
	}
	return results, nil
}

func (r *jsonProductRepository) Create(p *model.Product) (*model.Product, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	maxID := 0
	for _, product := range all {
		if product.ID > maxID {
			maxID = product.ID
		}
	}
	p.ID = maxID + 1
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	p.IsActive = true
	all = append(all, *p)
	return p, r.save(all)
}

func (r *jsonProductRepository) Update(id int, updated *model.Product) (*model.Product, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			updated.ID = id
			updated.CreatedAt = all[i].CreatedAt
			updated.UpdatedAt = time.Now()
			updated.IsActive = all[i].IsActive
			updated.ViewCount = all[i].ViewCount
			all[i] = *updated
			return &all[i], r.save(all)
		}
	}
	return nil, errors.New("product not found")
}

func (r *jsonProductRepository) Delete(id int) error {
	all, err := r.load()
	if err != nil {
		return err
	}
	for i := range all {
		if all[i].ID == id {
			all = append(all[:i], all[i+1:]...)
			return r.save(all)
		}
	}
	return errors.New("product not found")
}

// containsIgnoreCase checks if s contains substr case-insensitively.
func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	sLower := []byte(s)
	subLower := []byte(substr)
	toLower := func(b []byte) {
		for i, c := range b {
			if c >= 'A' && c <= 'Z' {
				b[i] = c + 32
			}
		}
	}
	toLower(sLower)
	toLower(subLower)
	return contains(string(sLower), string(subLower))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		})())
}
