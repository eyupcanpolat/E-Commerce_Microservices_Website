// Package tests contains unit tests for ProductService business logic.
package tests

import (
	"errors"
	"strings"
	"testing"

	"eticaret/product-service/internal/model"
	"eticaret/product-service/internal/service"
)

// ── Mock Repository ───────────────────────────────────────────────────────────

type mockProductRepository struct {
	products []model.Product
	nextID   int
}

func newMockProductRepo() *mockProductRepository {
	inStock := "in_stock"
	categoryID := 1
	salePrice := 9.99
	return &mockProductRepository{
		nextID: 4,
		products: []model.Product{
			{ID: 1, Name: "Laptop", Slug: "laptop", Price: 1000.0, StockStatus: inStock, StockQuantity: 50, IsActive: true, IsFeatured: true, CategoryID: &categoryID},
			{ID: 2, Name: "Mouse", Slug: "mouse", Price: 50.0, StockStatus: inStock, StockQuantity: 5, IsActive: true, IsFeatured: false, CategoryID: &categoryID},
			{ID: 3, Name: "Keyboard", Slug: "klavye", Description: "mechanical keyboard", Price: 200.0, SalePrice: &salePrice, StockStatus: "out_of_stock", StockQuantity: 0, IsActive: true},
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
		if filter.Search != "" && !strings.Contains(strings.ToLower(p.Name), strings.ToLower(filter.Search)) {
			continue
		}
		filtered = append(filtered, p)
	}
	return &model.PaginatedProducts{Data: filtered, Total: len(filtered), Page: page, PerPage: perPage, TotalPages: 1}, nil
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
	q := strings.ToLower(query)
	for _, p := range m.products {
		if p.IsActive && (strings.Contains(strings.ToLower(p.Name), q) || strings.Contains(strings.ToLower(p.Description), q)) {
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
	for i := range m.products {
		if m.products[i].ID == id {
			p.ID = id
			m.products[i] = *p
			return &m.products[i], nil
		}
	}
	return nil, errors.New("not found")
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

// ── GetProduct testleri ───────────────────────────────────────────────────────

func TestGetProduct_Found(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	product, err := svc.GetProduct(1)
	if err != nil {
		t.Fatalf("beklenen hata yok, alınan: %v", err)
	}
	if product.Name != "Laptop" {
		t.Errorf("beklenen Laptop, alınan %s", product.Name)
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	_, err := svc.GetProduct(999)
	if err != service.ErrProductNotFound {
		t.Errorf("beklenen ErrProductNotFound, alınan %v", err)
	}
}

func TestGetProductBySlug_Found(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	product, err := svc.GetProductBySlug("laptop")
	if err != nil {
		t.Fatalf("beklenen hata yok, alınan: %v", err)
	}
	if product.ID != 1 {
		t.Errorf("beklenen ID 1, alınan %d", product.ID)
	}
}

func TestGetProductBySlug_NotFound(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	_, err := svc.GetProductBySlug("olmayan-slug")
	if err != service.ErrProductNotFound {
		t.Errorf("beklenen ErrProductNotFound, alınan %v", err)
	}
}

// ── GetFeaturedProducts testleri ──────────────────────────────────────────────

func TestGetFeaturedProducts(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	products, err := svc.GetFeaturedProducts(4)
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if len(products) != 1 {
		t.Errorf("beklenen 1 öne çıkan ürün, alınan %d", len(products))
	}
	if products[0].Name != "Laptop" {
		t.Errorf("beklenen Laptop, alınan %s", products[0].Name)
	}
}

func TestGetFeaturedProducts_DefaultLimit(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	// limit <= 0 verildiğinde 4 kullanılmalı, hata olmamalı
	products, err := svc.GetFeaturedProducts(0)
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if products == nil {
		t.Error("sonuç nil olmamalı")
	}
}

// ── ListProducts testleri ─────────────────────────────────────────────────────

func TestListProducts_All(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	result, err := svc.ListProducts(model.ProductFilter{}, 1, 12)
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("beklenen 3 ürün, alınan %d", result.Total)
	}
}

func TestListProducts_InStockFilter(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	result, err := svc.ListProducts(model.ProductFilter{InStock: true}, 1, 12)
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	// Keyboard out_of_stock — filtrelenmeli
	if result.Total != 2 {
		t.Errorf("beklenen 2 stokta ürün, alınan %d", result.Total)
	}
}

func TestListProducts_DefaultPerPage(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	// perPage 0 verildiğinde 12 kullanılmalı
	result, err := svc.ListProducts(model.ProductFilter{}, 1, 0)
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if result.PerPage != 12 {
		t.Errorf("beklenen perPage 12, alınan %d", result.PerPage)
	}
}

// ── SearchProducts testleri ───────────────────────────────────────────────────

func TestSearchProducts_EmptyQuery(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	results, err := svc.SearchProducts("")
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("boş sorgu için 0 sonuç beklendi, alınan %d", len(results))
	}
}

func TestSearchProducts_WithQuery(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	results, err := svc.SearchProducts("keyboard")
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if len(results) == 0 {
		t.Error("keyboard araması sonuç döndürmeli")
	}
}

func TestSearchProducts_WhitespaceQuery(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	// Sadece boşluk → boş sonuç
	results, err := svc.SearchProducts("   ")
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("boşluk sorgusu için 0 sonuç beklendi, alınan %d", len(results))
	}
}

// ── CreateProduct testleri ────────────────────────────────────────────────────

func TestCreateProduct_Success(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	product, err := svc.CreateProduct(model.CreateProductRequest{
		Name:          "Yeni Ürün",
		Price:         299.99,
		StockQuantity: 50,
	})
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if product.StockStatus != "in_stock" {
		t.Errorf("beklenen in_stock, alınan %s", product.StockStatus)
	}
	if product.ID == 0 {
		t.Error("ürün ID atanmış olmalı")
	}
}

func TestCreateProduct_NameRequired(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	_, err := svc.CreateProduct(model.CreateProductRequest{Name: "", Price: 100.0})
	if err != service.ErrNameRequired {
		t.Errorf("beklenen ErrNameRequired, alınan %v", err)
	}
}

func TestCreateProduct_InvalidPrice(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	_, err := svc.CreateProduct(model.CreateProductRequest{Name: "Test", Price: -10.0})
	if err != service.ErrInvalidPrice {
		t.Errorf("beklenen ErrInvalidPrice, alınan %v", err)
	}
}

func TestCreateProduct_ZeroPrice(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	_, err := svc.CreateProduct(model.CreateProductRequest{Name: "Test", Price: 0})
	if err != service.ErrInvalidPrice {
		t.Errorf("sıfır fiyat için ErrInvalidPrice beklendi, alınan %v", err)
	}
}

func TestCreateProduct_StockStatus_OutOfStock(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	product, err := svc.CreateProduct(model.CreateProductRequest{
		Name:          "Stoksuz Ürün",
		Price:         100.0,
		StockQuantity: 0,
	})
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if product.StockStatus != "out_of_stock" {
		t.Errorf("beklenen out_of_stock, alınan %s", product.StockStatus)
	}
}

func TestCreateProduct_StockStatus_LowStock(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	product, err := svc.CreateProduct(model.CreateProductRequest{
		Name:          "Az Stoklu Ürün",
		Price:         100.0,
		StockQuantity: 5, // < 10
	})
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if product.StockStatus != "low_stock" {
		t.Errorf("beklenen low_stock, alınan %s", product.StockStatus)
	}
}

// ── UpdateProduct testleri ────────────────────────────────────────────────────

func TestUpdateProduct_Success(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	product, err := svc.UpdateProduct(1, model.CreateProductRequest{
		Name:          "Güncellenmiş Laptop",
		Price:         1200.0,
		StockQuantity: 30,
	})
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if product.Name != "Güncellenmiş Laptop" {
		t.Errorf("beklenen 'Güncellenmiş Laptop', alınan '%s'", product.Name)
	}
	if product.Price != 1200.0 {
		t.Errorf("beklenen fiyat 1200.0, alınan %f", product.Price)
	}
}

func TestUpdateProduct_NameRequired(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	_, err := svc.UpdateProduct(1, model.CreateProductRequest{Name: "", Price: 100.0})
	if err != service.ErrNameRequired {
		t.Errorf("beklenen ErrNameRequired, alınan %v", err)
	}
}

func TestUpdateProduct_InvalidPrice(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	_, err := svc.UpdateProduct(1, model.CreateProductRequest{Name: "Test", Price: 0})
	if err != service.ErrInvalidPrice {
		t.Errorf("beklenen ErrInvalidPrice, alınan %v", err)
	}
}

// ── DeleteProduct testleri ────────────────────────────────────────────────────

func TestDeleteProduct_Success(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	err := svc.DeleteProduct(1)
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}

	// Silinen ürün artık bulunamamalı
	_, err = svc.GetProduct(1)
	if err != service.ErrProductNotFound {
		t.Error("silinen ürün bulunamaz olmalı")
	}
}

func TestDeleteProduct_NotFound(t *testing.T) {
	svc := service.NewProductService(newMockProductRepo())

	err := svc.DeleteProduct(999)
	if err == nil {
		t.Error("olmayan ürün silinmemeli, hata beklendi")
	}
}
