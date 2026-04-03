// Package service contains business logic for OrderService.
// It also communicates with ProductService to validate products and prices.
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"eticaret/order-service/internal/model"
	"eticaret/order-service/internal/repository"
	"eticaret/shared/logger"
)

var (
	ErrOrderNotFound   = errors.New("sipariş bulunamadı")
	ErrAccessDenied    = errors.New("bu siparişe erişim yetkiniz yok")
	ErrNoItems         = errors.New("sipariş en az bir ürün içermelidir")
	ErrInvalidQuantity = errors.New("ürün miktarı 1 veya daha fazla olmalıdır")
	ErrProductUnavailable = errors.New("ürün stokta yok")
)

// OrderService defines all order-related operations.
type OrderService interface {
	GetUserOrders(userID int) ([]model.Order, error)
	GetOrder(id, userID int) (*model.Order, error)
	GetOrderByNumber(orderNumber string, userID int) (*model.Order, error)
	CreateOrder(userID int, req model.CreateOrderRequest) (*model.Order, error)
	CancelOrder(orderNumber string, userID int) error
	UpdateStatus(id int, status model.OrderStatus) (*model.Order, error) // admin only
}

type orderService struct {
	repo           repository.OrderRepository
	productSvcURL  string // URL of ProductService e.g. "http://localhost:8082"
	httpClient     *http.Client
}

// NewOrderService creates OrderService. productSvcURL is used for inter-service calls.
func NewOrderService(repo repository.OrderRepository, productSvcURL string) OrderService {
	return &orderService{
		repo:          repo,
		productSvcURL: productSvcURL,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *orderService) GetUserOrders(userID int) ([]model.Order, error) {
	orders, err := s.repo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	if orders == nil {
		orders = []model.Order{}
	}
	return orders, nil
}

func (s *orderService) GetOrder(id, userID int) (*model.Order, error) {
	order, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.UserID != userID {
		return nil, ErrAccessDenied
	}
	return order, nil
}

func (s *orderService) GetOrderByNumber(orderNumber string, userID int) (*model.Order, error) {
	order, err := s.repo.GetByOrderNumber(orderNumber)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.UserID != userID {
		return nil, ErrAccessDenied
	}
	return order, nil
}

// CreateOrder creates a new order, fetching product details from ProductService.
// This demonstrates inter-service HTTP communication.
func (s *orderService) CreateOrder(userID int, req model.CreateOrderRequest) (*model.Order, error) {
	if len(req.Items) == 0 {
		return nil, ErrNoItems
	}

	var orderItems []model.OrderItem
	var subtotal float64

	// Inter-service call: fetch product details from ProductService
	for _, item := range req.Items {
		if item.Quantity < 1 {
			return nil, ErrInvalidQuantity
		}

		product, err := s.fetchProduct(item.ProductID)
		if err != nil {
			logger.Warn("Failed to fetch product from product-service",
				"product_id", item.ProductID, "error", err)
			return nil, fmt.Errorf("ürün bilgisi alınamadı (id: %d)", item.ProductID)
		}

		if product.StockStatus == "out_of_stock" {
			return nil, fmt.Errorf("'%s' ürünü stokta yok", product.Name)
		}

		unitPrice := product.Price
		if product.SalePrice != nil && *product.SalePrice > 0 {
			unitPrice = *product.SalePrice
		}
		totalPrice := unitPrice * float64(item.Quantity)
		subtotal += totalPrice

		orderItems = append(orderItems, model.OrderItem{
			ProductID:   product.ID,
			ProductName: product.Name,
			ProductSKU:  product.SKU,
			Quantity:    item.Quantity,
			UnitPrice:   unitPrice,
			TotalPrice:  totalPrice,
		})
	}

	// Calculate shipping and tax
	shippingCost := 0.0
	if subtotal < 500 {
		shippingCost = 29.90
	}
	tax := subtotal * 0.18 // 18% KDV

	order := &model.Order{
		UserID:            userID,
		ShippingAddressID: req.ShippingAddressID,
		ShippingMethod:    req.ShippingMethod,
		PaymentMethod:     req.PaymentMethod,
		Notes:             req.Notes,
		Items:             orderItems,
		Subtotal:          subtotal,
		ShippingCost:      shippingCost,
		Tax:               tax,
		Total:             subtotal + shippingCost + tax,
	}

	return s.repo.Create(order)
}

func (s *orderService) CancelOrder(orderNumber string, userID int) error {
	order, err := s.repo.GetByOrderNumber(orderNumber)
	if err != nil {
		return ErrOrderNotFound
	}
	if order.UserID != userID {
		return ErrAccessDenied
	}
	return s.repo.Cancel(order.ID, userID)
}

func (s *orderService) UpdateStatus(id int, status model.OrderStatus) (*model.Order, error) {
	return s.repo.UpdateStatus(id, status)
}

// productResponse is the minimal struct we need from ProductService's response.
type productResponse struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	SKU         string   `json:"sku"`
	Price       float64  `json:"price"`
	SalePrice   *float64 `json:"sale_price"`
	StockStatus string   `json:"stock_status"`
}

// fetchProduct calls ProductService to get product details.
// This is the inter-service HTTP communication pattern.
func (s *orderService) fetchProduct(productID int) (*productResponse, error) {
	url := fmt.Sprintf("%s/products/%d", s.productSvcURL, productID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("product-service request creation failed: %w", err)
	}

	secret := os.Getenv("INTERNAL_SECRET")
	if secret == "" {
		secret = "internal-gateway-secret-change-in-prod" // default
	}
	req.Header.Set("X-Internal-Secret", secret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("product-service isteği başarısız: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product-service %d döndürdü", resp.StatusCode)
	}

	var envelope struct {
		Success bool            `json:"success"`
		Data    productResponse `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("product-service yanıtı parse edilemedi: %w", err)
	}

	// Fallback: if product service is down, use env-based mock
	if envelope.Data.ID == 0 {
		return nil, errors.New("ürün bulunamadı")
	}

	return &envelope.Data, nil
}

// fetchProductFromEnv is a fallback when PRODUCT_SERVICE_URL is not set.
// Reads directly from the product JSON file (for local dev without Docker).
func fetchProductFromFile(productID int) (*productResponse, error) {
	dataPath := os.Getenv("PRODUCT_DATA_PATH")
	if dataPath == "" {
		return nil, errors.New("PRODUCT_DATA_PATH not configured")
	}

	data, err := os.ReadFile(dataPath)
	if err != nil {
		return nil, err
	}

	var products []productResponse
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, err
	}

	for _, p := range products {
		if p.ID == productID {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("product %d not found", productID)
}
