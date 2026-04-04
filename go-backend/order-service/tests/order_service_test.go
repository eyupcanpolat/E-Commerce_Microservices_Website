// Package tests contains unit tests for OrderService business logic.
// Uses mock repository to avoid file I/O; tests ownership isolation and status transitions.
package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"eticaret/order-service/internal/model"
	"eticaret/order-service/internal/service"
)

// ── Mock OrderRepository ────────────────────────────────────────────────────

type mockOrderRepository struct {
	mu     sync.RWMutex
	orders []model.Order
	nextID int
}

func newMockOrderRepo() *mockOrderRepository {
	return &mockOrderRepository{nextID: 1}
}

func (m *mockOrderRepository) GetByUserID(userID int) ([]model.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []model.Order
	for _, o := range m.orders {
		if o.UserID == userID {
			result = append(result, o)
		}
	}
	return result, nil
}

func (m *mockOrderRepository) GetByID(id int) (*model.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := range m.orders {
		if m.orders[i].ID == id {
			return &m.orders[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockOrderRepository) GetByOrderNumber(orderNumber string) (*model.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := range m.orders {
		if m.orders[i].OrderNumber == orderNumber {
			return &m.orders[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockOrderRepository) Create(order *model.Order) (*model.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order.ID = m.nextID
	m.nextID++
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	order.Status = model.StatusPending
	order.PaymentStatus = "pending"
	order.OrderNumber = fmt.Sprintf("ORD-%d-%04d", time.Now().Year(), order.ID)
	for i := range order.Items {
		order.Items[i].ID = i + 1
		order.Items[i].OrderID = order.ID
	}
	m.orders = append(m.orders, *order)
	return order, nil
}

func (m *mockOrderRepository) UpdateStatus(id int, status model.OrderStatus) (*model.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.orders {
		if m.orders[i].ID == id {
			m.orders[i].Status = status
			m.orders[i].UpdatedAt = time.Now()
			now := time.Now()
			if status == model.StatusShipped {
				m.orders[i].ShippedAt = &now
			}
			if status == model.StatusDelivered {
				m.orders[i].DeliveredAt = &now
			}
			return &m.orders[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockOrderRepository) Cancel(id, userID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.orders {
		if m.orders[i].ID == id && m.orders[i].UserID == userID {
			s := m.orders[i].Status
			if s != model.StatusPending && s != model.StatusProcessing {
				return errors.New("bu aşamadaki sipariş iptal edilemez")
			}
			m.orders[i].Status = model.StatusCancelled
			m.orders[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return errors.New("not found or access denied")
}

// ── mockOrderService wraps the real service with a fixed productSvcURL ──────

// seedOrder creates an order directly in the repo (bypasses product fetch)
func seedOrder(repo *mockOrderRepository, userID int) *model.Order {
	order := &model.Order{
		UserID:  userID,
		Items:   []model.OrderItem{{ProductID: 1, ProductName: "Test Ürün", Quantity: 1, UnitPrice: 100, TotalPrice: 100}},
		Subtotal: 100, ShippingCost: 29.90, Tax: 18, Total: 147.90,
	}
	created, _ := repo.Create(order)
	return created
}

// ── Tests ───────────────────────────────────────────────────────────────────

func TestGetUserOrders_IsolatedByUser(t *testing.T) {
	repo := newMockOrderRepo()

	// Seed orders for two different users
	seedOrder(repo, 1)
	seedOrder(repo, 1)
	seedOrder(repo, 2)

	// We test directly through the repo since CreateOrder calls product-service
	orders1, _ := repo.GetByUserID(1)
	orders2, _ := repo.GetByUserID(2)

	if len(orders1) != 2 {
		t.Errorf("user 1 should have 2 orders, got %d", len(orders1))
	}
	if len(orders2) != 1 {
		t.Errorf("user 2 should have 1 order, got %d", len(orders2))
	}
}

func TestGetOrder_OwnershipCheck(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "") // empty productSvcURL (no inter-service calls needed)

	order := seedOrder(repo, 1)

	// Owner can access
	got, err := svc.GetOrder(order.ID, 1)
	if err != nil {
		t.Fatalf("owner should access order: %v", err)
	}
	if got.ID != order.ID {
		t.Errorf("expected ID %d, got %d", order.ID, got.ID)
	}

	// Other user cannot access
	_, err = svc.GetOrder(order.ID, 99)
	if err != service.ErrAccessDenied {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestGetOrder_NotFound(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "")

	_, err := svc.GetOrder(9999, 1)
	if err != service.ErrOrderNotFound {
		t.Errorf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestGetOrderByNumber_OwnershipCheck(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "")

	order := seedOrder(repo, 5)

	// Owner can fetch by number
	got, err := svc.GetOrderByNumber(order.OrderNumber, 5)
	if err != nil {
		t.Fatalf("owner should get by number: %v", err)
	}
	if got.OrderNumber != order.OrderNumber {
		t.Errorf("order number mismatch")
	}

	// Other user cannot
	_, err = svc.GetOrderByNumber(order.OrderNumber, 6)
	if err != service.ErrAccessDenied {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestUpdateStatus_Transitions(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "")

	order := seedOrder(repo, 1)

	tests := []struct {
		status model.OrderStatus
	}{
		{model.StatusProcessing},
		{model.StatusShipped},
		{model.StatusDelivered},
	}

	for _, tt := range tests {
		updated, err := svc.UpdateStatus(order.ID, tt.status)
		if err != nil {
			t.Fatalf("UpdateStatus(%s) failed: %v", tt.status, err)
		}
		if updated.Status != tt.status {
			t.Errorf("expected status %s, got %s", tt.status, updated.Status)
		}
	}
}

func TestUpdateStatus_ShippedAt_Set(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "")

	order := seedOrder(repo, 1)
	updated, err := svc.UpdateStatus(order.ID, model.StatusShipped)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ShippedAt == nil {
		t.Error("expected ShippedAt to be set when status is shipped")
	}
}

func TestUpdateStatus_DeliveredAt_Set(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "")

	order := seedOrder(repo, 1)
	updated, err := svc.UpdateStatus(order.ID, model.StatusDelivered)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.DeliveredAt == nil {
		t.Error("expected DeliveredAt to be set when status is delivered")
	}
}

func TestCancelOrder_Success(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "")

	order := seedOrder(repo, 1) // status = pending

	err := svc.CancelOrder(order.OrderNumber, 1)
	if err != nil {
		t.Fatalf("expected no error cancelling pending order: %v", err)
	}

	// Verify status changed
	got, _ := repo.GetByID(order.ID)
	if got.Status != model.StatusCancelled {
		t.Errorf("expected cancelled, got %s", got.Status)
	}
}

func TestCancelOrder_WrongUser(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "")

	order := seedOrder(repo, 1)

	err := svc.CancelOrder(order.OrderNumber, 42)
	if err != service.ErrAccessDenied {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestCancelOrder_NotFound(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "")

	err := svc.CancelOrder("ORD-INVALID-0000", 1)
	if err != service.ErrOrderNotFound {
		t.Errorf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestCreateOrder_EmptyItems(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "")

	_, err := svc.CreateOrder(1, model.CreateOrderRequest{
		Items: []model.OrderItemRequest{}, // empty
	})

	if err != service.ErrNoItems {
		t.Errorf("expected ErrNoItems, got %v", err)
	}
}

func TestCreateOrder_InvalidQuantity(t *testing.T) {
	repo := newMockOrderRepo()
	// Use empty product service URL — will fail on HTTP call but quantity validated first
	svc := service.NewOrderService(repo, "http://localhost:9999")

	_, err := svc.CreateOrder(1, model.CreateOrderRequest{
		Items: []model.OrderItemRequest{
			{ProductID: 1, Quantity: 0}, // invalid
		},
	})

	if err != service.ErrInvalidQuantity {
		t.Errorf("expected ErrInvalidQuantity, got %v", err)
	}
}

func TestOrderNumber_Format(t *testing.T) {
	repo := newMockOrderRepo()
	order := seedOrder(repo, 1)

	year := time.Now().Year()
	expected := fmt.Sprintf("ORD-%d-0001", year)
	if order.OrderNumber != expected {
		t.Errorf("expected order number format %s, got %s", expected, order.OrderNumber)
	}
}

func TestGetUserOrders_ReturnsEmpty(t *testing.T) {
	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, "")

	orders, err := svc.GetUserOrders(999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("expected 0 orders for unknown user, got %d", len(orders))
	}
}

// ── CreateOrder + fake product-service testleri ───────────────────────────────

// fakeProductServer sahte bir product-service başlatır.
func fakeProductServer(price float64, stockStatus string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"id":           1,
				"name":         "Test Ürün",
				"sku":          "TST-001",
				"price":        price,
				"sale_price":   nil,
				"stock_status": stockStatus,
			},
		})
	}))
}

func TestCreateOrder_Success(t *testing.T) {
	srv := fakeProductServer(100.0, "in_stock")
	defer srv.Close()

	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, srv.URL)

	order, err := svc.CreateOrder(1, model.CreateOrderRequest{
		Items: []model.OrderItemRequest{{ProductID: 1, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if order.ID == 0 {
		t.Error("sipariş ID atanmış olmalı")
	}
	if order.UserID != 1 {
		t.Errorf("beklenen userID 1, alınan %d", order.UserID)
	}
	if order.Subtotal != 200.0 { // 2 x 100
		t.Errorf("beklenen subtotal 200, alınan %f", order.Subtotal)
	}
}

func TestCreateOrder_ShippingCost_PaidUnder500(t *testing.T) {
	srv := fakeProductServer(100.0, "in_stock")
	defer srv.Close()

	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, srv.URL)

	order, err := svc.CreateOrder(1, model.CreateOrderRequest{
		Items: []model.OrderItemRequest{{ProductID: 1, Quantity: 1}}, // 100 TL < 500
	})
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if order.ShippingCost != 29.90 {
		t.Errorf("500 TL altı için kargo 29.90 olmalı, alınan %f", order.ShippingCost)
	}
}

func TestCreateOrder_ShippingCost_FreeOver500(t *testing.T) {
	srv := fakeProductServer(600.0, "in_stock")
	defer srv.Close()

	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, srv.URL)

	order, err := svc.CreateOrder(1, model.CreateOrderRequest{
		Items: []model.OrderItemRequest{{ProductID: 1, Quantity: 1}}, // 600 TL >= 500
	})
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	if order.ShippingCost != 0 {
		t.Errorf("500 TL üstü için kargo ücretsiz olmalı, alınan %f", order.ShippingCost)
	}
}

func TestCreateOrder_TaxCalculation(t *testing.T) {
	srv := fakeProductServer(100.0, "in_stock")
	defer srv.Close()

	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, srv.URL)

	order, err := svc.CreateOrder(1, model.CreateOrderRequest{
		Items: []model.OrderItemRequest{{ProductID: 1, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("beklenmedik hata: %v", err)
	}
	expectedTax := 100.0 * 0.18 // %18 KDV
	if order.Tax != expectedTax {
		t.Errorf("beklenen KDV %f, alınan %f", expectedTax, order.Tax)
	}
}

func TestCreateOrder_ProductOutOfStock(t *testing.T) {
	srv := fakeProductServer(100.0, "out_of_stock")
	defer srv.Close()

	repo := newMockOrderRepo()
	svc := service.NewOrderService(repo, srv.URL)

	_, err := svc.CreateOrder(1, model.CreateOrderRequest{
		Items: []model.OrderItemRequest{{ProductID: 1, Quantity: 1}},
	})
	if err == nil {
		t.Error("stokta olmayan ürün için hata beklendi")
	}
}
