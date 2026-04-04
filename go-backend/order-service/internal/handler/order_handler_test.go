package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"eticaret/order-service/internal/handler"
	"eticaret/order-service/internal/model"
	"eticaret/order-service/internal/service"
)

// ── Mock OrderService ─────────────────────────────────────────────────────────

type mockOrderService struct {
	getUserOrdersFn   func(userID int) ([]model.Order, error)
	getOrderFn        func(id, userID int) (*model.Order, error)
	getOrderByNumFn   func(orderNumber string, userID int) (*model.Order, error)
	createOrderFn     func(userID int, req model.CreateOrderRequest) (*model.Order, error)
	cancelOrderFn     func(orderNumber string, userID int) error
	updateStatusFn    func(id int, status model.OrderStatus) (*model.Order, error)
}

func (m *mockOrderService) GetUserOrders(userID int) ([]model.Order, error) {
	return m.getUserOrdersFn(userID)
}
func (m *mockOrderService) GetOrder(id, userID int) (*model.Order, error) {
	return m.getOrderFn(id, userID)
}
func (m *mockOrderService) GetOrderByNumber(orderNumber string, userID int) (*model.Order, error) {
	return m.getOrderByNumFn(orderNumber, userID)
}
func (m *mockOrderService) CreateOrder(userID int, req model.CreateOrderRequest) (*model.Order, error) {
	return m.createOrderFn(userID, req)
}
func (m *mockOrderService) CancelOrder(orderNumber string, userID int) error {
	return m.cancelOrderFn(orderNumber, userID)
}
func (m *mockOrderService) UpdateStatus(id int, status model.OrderStatus) (*model.Order, error) {
	return m.updateStatusFn(id, status)
}

// ── Yardımcılar ───────────────────────────────────────────────────────────────

func toJSON(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("JSON marshal hatası: %v", err)
	}
	return bytes.NewBuffer(b)
}

func fakeOrder() *model.Order {
	return &model.Order{
		ID:          1,
		UserID:      1,
		OrderNumber: "ORD-2026-0001",
		Status:      model.StatusPending,
		Subtotal:    100.0,
		Total:       147.90,
	}
}

func newMux(pattern string, fn http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(pattern, fn)
	return mux
}

// ── GetOrders handler testleri ────────────────────────────────────────────────

func TestGetOrdersHandler_Success(t *testing.T) {
	svc := &mockOrderService{
		getUserOrdersFn: func(userID int) ([]model.Order, error) {
			return []model.Order{*fakeOrder()}, nil
		},
	}
	h := handler.NewOrderHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.GetOrders(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestGetOrdersHandler_ReturnsEmpty(t *testing.T) {
	svc := &mockOrderService{
		getUserOrdersFn: func(userID int) ([]model.Order, error) {
			return []model.Order{}, nil
		},
	}
	h := handler.NewOrderHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.GetOrders(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

// ── GetOrder handler testleri ─────────────────────────────────────────────────

func TestGetOrderHandler_Success(t *testing.T) {
	svc := &mockOrderService{
		getOrderFn: func(id, userID int) (*model.Order, error) {
			return fakeOrder(), nil
		},
	}
	h := handler.NewOrderHandler(svc)

	mux := newMux("GET /orders/{id}", h.GetOrder)
	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestGetOrderHandler_InvalidID(t *testing.T) {
	svc := &mockOrderService{}
	h := handler.NewOrderHandler(svc)

	mux := newMux("GET /orders/{id}", h.GetOrder)
	req := httptest.NewRequest(http.MethodGet, "/orders/abc", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestGetOrderHandler_NotFound(t *testing.T) {
	svc := &mockOrderService{
		getOrderFn: func(id, userID int) (*model.Order, error) {
			return nil, service.ErrOrderNotFound
		},
	}
	h := handler.NewOrderHandler(svc)

	mux := newMux("GET /orders/{id}", h.GetOrder)
	req := httptest.NewRequest(http.MethodGet, "/orders/999", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("beklenen 404, alınan %d", rr.Code)
	}
}

func TestGetOrderHandler_AccessDenied(t *testing.T) {
	svc := &mockOrderService{
		getOrderFn: func(id, userID int) (*model.Order, error) {
			return nil, service.ErrAccessDenied
		},
	}
	h := handler.NewOrderHandler(svc)

	mux := newMux("GET /orders/{id}", h.GetOrder)
	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	req.Header.Set("X-User-ID", "99")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}

// ── CreateOrder handler testleri ──────────────────────────────────────────────

func TestCreateOrderHandler_Success(t *testing.T) {
	svc := &mockOrderService{
		createOrderFn: func(userID int, req model.CreateOrderRequest) (*model.Order, error) {
			return fakeOrder(), nil
		},
	}
	h := handler.NewOrderHandler(svc)

	body := toJSON(t, map[string]interface{}{
		"items": []map[string]interface{}{
			{"product_id": 1, "quantity": 2},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", body)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.CreateOrder(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("beklenen 201, alınan %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateOrderHandler_InvalidJSON(t *testing.T) {
	svc := &mockOrderService{}
	h := handler.NewOrderHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString("bad json"))
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.CreateOrder(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestCreateOrderHandler_NoItems(t *testing.T) {
	svc := &mockOrderService{
		createOrderFn: func(userID int, req model.CreateOrderRequest) (*model.Order, error) {
			return nil, service.ErrNoItems
		},
	}
	h := handler.NewOrderHandler(svc)

	body := toJSON(t, map[string]interface{}{"items": []interface{}{}})
	req := httptest.NewRequest(http.MethodPost, "/orders", body)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.CreateOrder(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

// ── CancelOrder handler testleri ──────────────────────────────────────────────

func TestCancelOrderHandler_Success(t *testing.T) {
	svc := &mockOrderService{
		cancelOrderFn: func(orderNumber string, userID int) error { return nil },
	}
	h := handler.NewOrderHandler(svc)

	mux := newMux("POST /orders/{orderNumber}/cancel", h.CancelOrder)
	req := httptest.NewRequest(http.MethodPost, "/orders/ORD-2026-0001/cancel", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestCancelOrderHandler_AccessDenied(t *testing.T) {
	svc := &mockOrderService{
		cancelOrderFn: func(orderNumber string, userID int) error {
			return service.ErrAccessDenied
		},
	}
	h := handler.NewOrderHandler(svc)

	mux := newMux("POST /orders/{orderNumber}/cancel", h.CancelOrder)
	req := httptest.NewRequest(http.MethodPost, "/orders/ORD-2026-0001/cancel", nil)
	req.Header.Set("X-User-ID", "99")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}

// ── UpdateStatus handler testleri ─────────────────────────────────────────────

func TestUpdateStatusHandler_Success(t *testing.T) {
	svc := &mockOrderService{
		updateStatusFn: func(id int, status model.OrderStatus) (*model.Order, error) {
			o := fakeOrder()
			o.Status = status
			return o, nil
		},
	}
	h := handler.NewOrderHandler(svc)

	body := toJSON(t, map[string]string{"status": "shipped"})
	mux := newMux("PUT /orders/{id}/status", h.UpdateStatus)
	req := httptest.NewRequest(http.MethodPut, "/orders/1/status", body)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestUpdateStatusHandler_InvalidID(t *testing.T) {
	svc := &mockOrderService{}
	h := handler.NewOrderHandler(svc)

	body := toJSON(t, map[string]string{"status": "shipped"})
	mux := newMux("PUT /orders/{id}/status", h.UpdateStatus)
	req := httptest.NewRequest(http.MethodPut, "/orders/abc/status", body)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestUpdateStatusHandler_InvalidJSON(t *testing.T) {
	svc := &mockOrderService{}
	h := handler.NewOrderHandler(svc)

	mux := newMux("PUT /orders/{id}/status", h.UpdateStatus)
	req := httptest.NewRequest(http.MethodPut, "/orders/1/status", bytes.NewBufferString("bad json"))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

// ── Health handler testi ──────────────────────────────────────────────────────

func TestHealthHandler_Success(t *testing.T) {
	svc := &mockOrderService{}
	h := handler.NewOrderHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}
