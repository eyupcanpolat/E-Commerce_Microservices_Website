// Package handler - product handlers using gateway-injected identity headers
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"eticaret/product-service/internal/middleware"
	"eticaret/product-service/internal/model"
	"eticaret/product-service/internal/service"
	"eticaret/shared/logger"
	"eticaret/shared/response"
)

type ProductHandler struct {
	productService service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{productService: svc}
}

// ListProducts handles GET /products — public
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	filter := model.ProductFilter{
		Search: q.Get("q"),
		Sort:   q.Get("sort"),
	}

	if catID := q.Get("category"); catID != "" {
		id, err := strconv.Atoi(catID)
		if err == nil {
			filter.CategoryID = &id
		}
	}
	if minP := q.Get("min_price"); minP != "" {
		v, err := strconv.ParseFloat(minP, 64)
		if err == nil {
			filter.MinPrice = &v
		}
	}
	if maxP := q.Get("max_price"); maxP != "" {
		v, err := strconv.ParseFloat(maxP, 64)
		if err == nil {
			filter.MaxPrice = &v
		}
	}
	filter.InStock = q.Get("in_stock") == "1" || strings.ToLower(q.Get("in_stock")) == "true"

	result, err := h.productService.ListProducts(filter, page, 12)
	if err != nil {
		logger.Error("ListProducts failed", "error", err)
		response.InternalServerError(w, "")
		return
	}
	response.Success(w, "", result)
}

// GetProduct handles GET /products/{id}
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Geçersiz ürün ID")
		return
	}
	product, err := h.productService.GetProduct(id)
	if err != nil {
		response.NotFound(w, "Ürün bulunamadı")
		return
	}
	response.Success(w, "", product)
}

// GetProductBySlug handles GET /products/slug/{slug}
func (h *ProductHandler) GetProductBySlug(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	product, err := h.productService.GetProductBySlug(slug)
	if err != nil {
		response.NotFound(w, "Ürün bulunamadı")
		return
	}
	response.Success(w, "", product)
}

// GetFeatured handles GET /products/featured
func (h *ProductHandler) GetFeatured(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 4
	}
	products, err := h.productService.GetFeaturedProducts(limit)
	if err != nil {
		response.InternalServerError(w, "")
		return
	}
	response.Success(w, "", products)
}

// Search handles GET /products/search?q=...
func (h *ProductHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	products, err := h.productService.SearchProducts(query)
	if err != nil {
		response.InternalServerError(w, "")
		return
	}
	response.Success(w, "", map[string]interface{}{
		"query":   query,
		"count":   len(products),
		"results": products,
	})
}

// CreateProduct handles POST /products — admin only (enforced at gateway)
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Trust gateway: X-User-Role was verified at gateway level
	userEmail := middleware.GetUserEmail(r)

	var req model.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Geçersiz JSON")
		return
	}

	product, err := h.productService.CreateProduct(req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	logger.Info("Product created", "id", product.ID, "name", product.Name, "by", userEmail)
	response.Created(w, "Ürün oluşturuldu", product)
}

// UpdateProduct handles PUT /products/{id} — admin only
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Geçersiz ürün ID")
		return
	}

	var req model.CreateProductRequest // can reuse create struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Geçersiz JSON")
		return
	}

	product, err := h.productService.UpdateProduct(id, req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	response.Success(w, "Ürün güncellendi", product)
}

// DeleteProduct handles DELETE /products/{id} — admin only (enforced at gateway)
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Geçersiz ürün ID")
		return
	}
	if err := h.productService.DeleteProduct(id); err != nil {
		response.NotFound(w, "Ürün bulunamadı")
		return
	}
	response.Success(w, "Ürün silindi", nil)
}

// Health handles GET /health
func (h *ProductHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.Success(w, "product-service is healthy", map[string]string{
		"service":           "product-service",
		"status":            "ok",
		"network_isolation": "active",
	})
}
