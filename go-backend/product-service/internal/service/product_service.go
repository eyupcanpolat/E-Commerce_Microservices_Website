// Package service contains business logic for ProductService.
package service

import (
	"errors"
	"strings"

	"eticaret/product-service/internal/model"
	"eticaret/product-service/internal/repository"
)

var (
	ErrProductNotFound = errors.New("ürün bulunamadı")
	ErrInvalidPrice    = errors.New("fiyat sıfırdan büyük olmalıdır")
	ErrNameRequired    = errors.New("ürün adı zorunludur")
)

// ProductService defines all business operations for products.
type ProductService interface {
	ListProducts(filter model.ProductFilter, page, perPage int) (*model.PaginatedProducts, error)
	GetProduct(id int) (*model.Product, error)
	GetProductBySlug(slug string) (*model.Product, error)
	GetFeaturedProducts(limit int) ([]model.Product, error)
	SearchProducts(query string) ([]model.Product, error)
	CreateProduct(req model.CreateProductRequest) (*model.Product, error)
	UpdateProduct(id int, req model.CreateProductRequest) (*model.Product, error)
	DeleteProduct(id int) error
}

type productService struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) ProductService {
	return &productService{repo: repo}
}

func (s *productService) ListProducts(filter model.ProductFilter, page, perPage int) (*model.PaginatedProducts, error) {
	if perPage <= 0 {
		perPage = 12
	}
	return s.repo.GetAll(filter, page, perPage)
}

func (s *productService) GetProduct(id int) (*model.Product, error) {
	p, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrProductNotFound
	}
	return p, nil
}

func (s *productService) GetProductBySlug(slug string) (*model.Product, error) {
	p, err := s.repo.GetBySlug(slug)
	if err != nil {
		return nil, ErrProductNotFound
	}
	return p, nil
}

func (s *productService) GetFeaturedProducts(limit int) ([]model.Product, error) {
	if limit <= 0 {
		limit = 4
	}
	return s.repo.GetFeatured(limit)
}

func (s *productService) SearchProducts(query string) ([]model.Product, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []model.Product{}, nil
	}
	return s.repo.Search(query)
}

func (s *productService) CreateProduct(req model.CreateProductRequest) (*model.Product, error) {
	// Validation
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrNameRequired
	}
	if req.Price <= 0 {
		return nil, ErrInvalidPrice
	}

	stockStatus := "in_stock"
	if req.StockQuantity == 0 {
		stockStatus = "out_of_stock"
	} else if req.StockQuantity < 10 {
		stockStatus = "low_stock"
	}

	p := &model.Product{
		CategoryID:       req.CategoryID,
		Name:             strings.TrimSpace(req.Name),
		Slug:             req.Slug,
		Description:      req.Description,
		ShortDescription: req.ShortDescription,
		Price:            req.Price,
		SalePrice:        req.SalePrice,
		SKU:              req.SKU,
		StockQuantity:    req.StockQuantity,
		StockStatus:      stockStatus,
		ImageURL:         req.ImageURL,
		GalleryImages:    req.GalleryImages,
		IsFeatured:       req.IsFeatured,
	}

	return s.repo.Create(p)
}

func (s *productService) UpdateProduct(id int, req model.CreateProductRequest) (*model.Product, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrNameRequired
	}
	if req.Price <= 0 {
		return nil, ErrInvalidPrice
	}

	stockStatus := "in_stock"
	if req.StockQuantity == 0 {
		stockStatus = "out_of_stock"
	} else if req.StockQuantity < 10 {
		stockStatus = "low_stock"
	}

	p := &model.Product{
		CategoryID:       req.CategoryID,
		Name:             strings.TrimSpace(req.Name),
		Slug:             req.Slug,
		Description:      req.Description,
		ShortDescription: req.ShortDescription,
		Price:            req.Price,
		SalePrice:        req.SalePrice,
		SKU:              req.SKU,
		StockQuantity:    req.StockQuantity,
		StockStatus:      stockStatus,
		ImageURL:         req.ImageURL,
		GalleryImages:    req.GalleryImages,
		IsFeatured:       req.IsFeatured,
	}

	return s.repo.Update(id, p)
}

func (s *productService) DeleteProduct(id int) error {
	return s.repo.Delete(id)
}
