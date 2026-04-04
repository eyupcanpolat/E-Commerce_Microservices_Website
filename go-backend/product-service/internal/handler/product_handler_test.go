package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"eticaret/product-service/internal/handler"
	"eticaret/product-service/internal/model"
	"eticaret/product-service/internal/service"
)

// ── Mock ProductService ───────────────────────────────────────────────────────

type mockProductService struct {
	listFn     func(filter model.ProductFilter, page, perPage int) (*model.PaginatedProducts, error)
	getFn      func(id int) (*model.Product, error)
	getSlugFn  func(slug string) (*model.Product, error)
	featuredFn func(limit int) ([]model.Product, error)
	searchFn   func(query string) ([]model.Product, error)
	createFn   func(req model.CreateProductRequest) (*model.Product, error)
	updateFn   func(id int, req model.CreateProductRequest) (*model.Product, error)
	deleteFn   func(id int) error
}

func (m *mockProductService) ListProducts(f model.ProductFilter, page, perPage int) (*model.PaginatedProducts, error) {
	return m.listFn(f, page, perPage)
}
func (m *mockProductService) GetProduct(id int) (*model.Product, error) { return m.getFn(id) }
func (m *mockProductService) GetProductBySlug(slug string) (*model.Product, error) {
	return m.getSlugFn(slug)
}
func (m *mockProductService) GetFeaturedProducts(limit int) ([]model.Product, error) {
	return m.featuredFn(limit)
}
func (m *mockProductService) SearchProducts(query string) ([]model.Product, error) {
	return m.searchFn(query)
}
func (m *mockProductService) CreateProduct(req model.CreateProductRequest) (*model.Product, error) {
	return m.createFn(req)
}
func (m *mockProductService) UpdateProduct(id int, req model.CreateProductRequest) (*model.Product, error) {
	return m.updateFn(id, req)
}
func (m *mockProductService) DeleteProduct(id int) error { return m.deleteFn(id) }

// ── Yardımcılar ───────────────────────────────────────────────────────────────

func toJSON(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("JSON marshal hatası: %v", err)
	}
	return bytes.NewBuffer(b)
}

func fakeProduct() *model.Product {
	return &model.Product{
		ID:          1,
		Name:        "Laptop",
		Slug:        "laptop",
		Price:       1000.0,
		StockStatus: "in_stock",
		IsActive:    true,
	}
}

// newHandler path value'ları destekleyen mux ile handler'ı sarmalar.
func newMux(pattern string, handlerFn http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(pattern, handlerFn)
	return mux
}

// ── ListProducts handler testleri ─────────────────────────────────────────────

func TestListProductsHandler_Success(t *testing.T) {
	svc := &mockProductService{
		listFn: func(f model.ProductFilter, page, perPage int) (*model.PaginatedProducts, error) {
			return &model.PaginatedProducts{
				Data:    []model.Product{*fakeProduct()},
				Total:   1,
				Page:    1,
				PerPage: 12,
			}, nil
		},
	}
	h := handler.NewProductHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestListProductsHandler_InStockFilter(t *testing.T) {
	var capturedFilter model.ProductFilter
	svc := &mockProductService{
		listFn: func(f model.ProductFilter, page, perPage int) (*model.PaginatedProducts, error) {
			capturedFilter = f
			return &model.PaginatedProducts{Data: []model.Product{}}, nil
		},
	}
	h := handler.NewProductHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/products?in_stock=1", nil)
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if !capturedFilter.InStock {
		t.Error("in_stock=1 filtresi service'e iletilmeli")
	}
}

// ── GetProduct handler testleri ───────────────────────────────────────────────

func TestGetProductHandler_Success(t *testing.T) {
	svc := &mockProductService{
		getFn: func(id int) (*model.Product, error) { return fakeProduct(), nil },
	}
	h := handler.NewProductHandler(svc)

	mux := newMux("GET /products/{id}", h.GetProduct)
	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestGetProductHandler_InvalidID(t *testing.T) {
	svc := &mockProductService{}
	h := handler.NewProductHandler(svc)

	mux := newMux("GET /products/{id}", h.GetProduct)
	req := httptest.NewRequest(http.MethodGet, "/products/abc", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestGetProductHandler_NotFound(t *testing.T) {
	svc := &mockProductService{
		getFn: func(id int) (*model.Product, error) { return nil, service.ErrProductNotFound },
	}
	h := handler.NewProductHandler(svc)

	mux := newMux("GET /products/{id}", h.GetProduct)
	req := httptest.NewRequest(http.MethodGet, "/products/999", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("beklenen 404, alınan %d", rr.Code)
	}
}

// ── GetFeatured handler testleri ──────────────────────────────────────────────

func TestGetFeaturedHandler_Success(t *testing.T) {
	svc := &mockProductService{
		featuredFn: func(limit int) ([]model.Product, error) {
			return []model.Product{*fakeProduct()}, nil
		},
	}
	h := handler.NewProductHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/products/featured", nil)
	rr := httptest.NewRecorder()
	h.GetFeatured(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

// ── Search handler testleri ───────────────────────────────────────────────────

func TestSearchHandler_Success(t *testing.T) {
	svc := &mockProductService{
		searchFn: func(query string) ([]model.Product, error) {
			return []model.Product{*fakeProduct()}, nil
		},
	}
	h := handler.NewProductHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/products/search?q=laptop", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["query"] != "laptop" {
		t.Errorf("beklenen query 'laptop', alınan %v", data["query"])
	}
}

// ── CreateProduct handler testleri ────────────────────────────────────────────

func TestCreateProductHandler_Success(t *testing.T) {
	svc := &mockProductService{
		createFn: func(req model.CreateProductRequest) (*model.Product, error) {
			return fakeProduct(), nil
		},
	}
	h := handler.NewProductHandler(svc)

	body := toJSON(t, map[string]interface{}{
		"name":  "Laptop",
		"price": 1000.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/products", body)
	rr := httptest.NewRecorder()
	h.CreateProduct(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("beklenen 201, alınan %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateProductHandler_InvalidJSON(t *testing.T) {
	svc := &mockProductService{}
	h := handler.NewProductHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString("bad json"))
	rr := httptest.NewRecorder()
	h.CreateProduct(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestCreateProductHandler_ValidationError(t *testing.T) {
	svc := &mockProductService{
		createFn: func(req model.CreateProductRequest) (*model.Product, error) {
			return nil, service.ErrNameRequired
		},
	}
	h := handler.NewProductHandler(svc)

	body := toJSON(t, map[string]interface{}{"name": "", "price": 100.0})
	req := httptest.NewRequest(http.MethodPost, "/products", body)
	rr := httptest.NewRecorder()
	h.CreateProduct(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

// ── DeleteProduct handler testleri ────────────────────────────────────────────

func TestDeleteProductHandler_Success(t *testing.T) {
	svc := &mockProductService{
		deleteFn: func(id int) error { return nil },
	}
	h := handler.NewProductHandler(svc)

	mux := newMux("DELETE /products/{id}", h.DeleteProduct)
	req := httptest.NewRequest(http.MethodDelete, "/products/1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestDeleteProductHandler_InvalidID(t *testing.T) {
	svc := &mockProductService{}
	h := handler.NewProductHandler(svc)

	mux := newMux("DELETE /products/{id}", h.DeleteProduct)
	req := httptest.NewRequest(http.MethodDelete, "/products/abc", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestDeleteProductHandler_NotFound(t *testing.T) {
	svc := &mockProductService{
		deleteFn: func(id int) error { return service.ErrProductNotFound },
	}
	h := handler.NewProductHandler(svc)

	mux := newMux("DELETE /products/{id}", h.DeleteProduct)
	req := httptest.NewRequest(http.MethodDelete, "/products/999", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("beklenen 404, alınan %d", rr.Code)
	}
}

// ── Health handler testi ──────────────────────────────────────────────────────

func TestHealthHandler_Success(t *testing.T) {
	svc := &mockProductService{}
	h := handler.NewProductHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}
