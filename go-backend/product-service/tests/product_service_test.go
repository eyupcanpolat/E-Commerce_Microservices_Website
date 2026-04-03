// Package tests contains unit tests for ProductService.
package tests

import (
	"errors"
	"testing"

	"eticaret/product-service/internal/model"
	"eticaret/product-service/internal/service"
)

// --- Mock Repository ---

type mockProductRepository struct {
	products []model.Product
	nextID   int
}

func newMockProductRepo() *mockProductRepository {
	in_stock := "in_stock"
	categoryID := 1
	salePrice := 9.99
	return &mockProductRepository{
		nextID: 4,
		products: []model.Product{
			{ID: 1, Name: "Laptop", Slug: "laptop", Price: 1000.0, StockStatus: in_stock, IsActive: true, IsFeatured: true, CategoryID: &categoryID},
			{ID: 2, Name: "Mouse", Slug: "mouse", Price: 50.0, StockStatus: in_stock, IsActive: true, IsFeatured: false, CategoryID: &categoryID},
			{ID: 3, Name: "Keyboard", Slug: "klavye", Description: "mechanical keyboard", Price: 200.0, SalePrice: &salePrice, StockStatus: "out_of_stock", IsActive: true},
		},
	}
}

func (m *mockProductRepository) GetAll(filter model.ProductFilter, page, perPage int) (*model.PaginatedProducts, error) {
	var filtered []model.Product
	for _, p := range m.products {
		if !p.IsActive {
			continue
		}
		if filter.InStock && p.StockStatus == "out_of_stock" {
			continue
		}
		filtered = append(filtered, p)
	}
	return &model.PaginatedProducts{Data: filtered, Total: len(filtered), Page: 1, PerPage: perPage, TotalPages: 1}, nil
}

func (m *mockProductRepository) GetByID(id int) (*model.Product, error) {
	for i := range m.products {
		if m.products[i].ID == id {
			return &m.products[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockProductRepository) GetBySlug(slug string) (*model.Product, error) {
	for i := range m.products {
		if m.products[i].Slug == slug {
			return &m.products[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockProductRepository) GetFeatured(limit int) ([]model.Product, error) {
	var result []model.Product
	for _, p := range m.products {
		if p.IsFeatured && p.IsActive {
			result = append(result, p)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockProductRepository) Search(query string) ([]model.Product, error) {
	var result []model.Product
	for _, p := range m.products {
		if p.IsActive {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockProductRepository) Create(p *model.Product) (*model.Product, error) {
	p.ID = m.nextID
	m.nextID++
	m.products = append(m.products, *p)
	return p, nil
}

func (m *mockProductRepository) Update(id int, p *model.Product) (*model.Product, error) {
	return p, nil
}

func (m *mockProductRepository) Delete(id int) error {
	for i := range m.products {
		if m.products[i].ID == id {
			m.products = append(m.products[:i], m.products[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

// --- Tests ---

func TestGetProduct_Found(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	product, err := svc.GetProduct(1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if product.Name != "Laptop" {
		t.Errorf("expected Laptop, got %s", product.Name)
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	_, err := svc.GetProduct(999)
	if err != service.ErrProductNotFound {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

func TestGetProductBySlug(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	product, err := svc.GetProductBySlug("laptop")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if product.ID != 1 {
		t.Errorf("expected ID 1, got %d", product.ID)
	}
}

func TestGetFeaturedProducts(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	products, err := svc.GetFeaturedProducts(4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Mock data has 1 featured product
	if len(products) != 1 {
		t.Errorf("expected 1 featured product, got %d", len(products))
	}
}

func TestCreateProduct_InvalidPrice(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	req := model.CreateProductRequest{
		Name:  "Test Product",
		Price: -10.0,
	}

	_, err := svc.CreateProduct(req)
	if err != service.ErrInvalidPrice {
		t.Errorf("expected ErrInvalidPrice, got %v", err)
	}
}

func TestCreateProduct_NameRequired(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	req := model.CreateProductRequest{
		Name:  "",
		Price: 100.0,
	}

	_, err := svc.CreateProduct(req)
	if err != service.ErrNameRequired {
		t.Errorf("expected ErrNameRequired, got %v", err)
	}
}

func TestCreateProduct_Success(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	req := model.CreateProductRequest{
		Name:          "New Product",
		Price:         299.99,
		StockQuantity: 50,
	}

	product, err := svc.CreateProduct(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if product.StockStatus != "in_stock" {
		t.Errorf("expected in_stock, got %s", product.StockStatus)
	}
}

func TestSearchProducts_EmptyQuery(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	results, err := svc.SearchProducts("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty search should return empty (not all products)
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}
