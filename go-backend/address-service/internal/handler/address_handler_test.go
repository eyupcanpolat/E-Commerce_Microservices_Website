package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"eticaret/address-service/internal/handler"
	"eticaret/address-service/internal/model"
	"eticaret/address-service/internal/service"
)

// ── Mock AddressService ───────────────────────────────────────────────────────

type mockAddressService struct {
	getListFn  func(userID int) ([]model.Address, error)
	getFn      func(id, userID int) (*model.Address, error)
	createFn   func(userID int, req model.AddressRequest) (*model.Address, error)
	updateFn   func(id, userID int, req model.AddressRequest) (*model.Address, error)
	deleteFn   func(id, userID int) error
}

func (m *mockAddressService) GetUserAddresses(userID int) ([]model.Address, error) {
	return m.getListFn(userID)
}
func (m *mockAddressService) GetAddress(id, userID int) (*model.Address, error) {
	return m.getFn(id, userID)
}
func (m *mockAddressService) CreateAddress(userID int, req model.AddressRequest) (*model.Address, error) {
	return m.createFn(userID, req)
}
func (m *mockAddressService) UpdateAddress(id, userID int, req model.AddressRequest) (*model.Address, error) {
	return m.updateFn(id, userID, req)
}
func (m *mockAddressService) DeleteAddress(id, userID int) error {
	return m.deleteFn(id, userID)
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

func fakeAddress() *model.Address {
	return &model.Address{
		ID:           1,
		UserID:       1,
		Title:        "Ev",
		AddressLine1: "Test Sokak No:1",
		City:         "İstanbul",
		PostalCode:   "34000",
		Country:      "Türkiye",
	}
}

func newMux(pattern string, fn http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(pattern, fn)
	return mux
}

// ── GetAddresses handler testleri ─────────────────────────────────────────────

func TestGetAddressesHandler_Success(t *testing.T) {
	svc := &mockAddressService{
		getListFn: func(userID int) ([]model.Address, error) {
			return []model.Address{*fakeAddress()}, nil
		},
	}
	h := handler.NewAddressHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/addresses", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.GetAddresses(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestGetAddressesHandler_ReturnsEmpty(t *testing.T) {
	svc := &mockAddressService{
		getListFn: func(userID int) ([]model.Address, error) {
			return nil, nil
		},
	}
	h := handler.NewAddressHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/addresses", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.GetAddresses(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

// ── GetAddress handler testleri ───────────────────────────────────────────────

func TestGetAddressHandler_Success(t *testing.T) {
	svc := &mockAddressService{
		getFn: func(id, userID int) (*model.Address, error) { return fakeAddress(), nil },
	}
	h := handler.NewAddressHandler(svc)

	mux := newMux("GET /addresses/{id}", h.GetAddress)
	req := httptest.NewRequest(http.MethodGet, "/addresses/1", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestGetAddressHandler_InvalidID(t *testing.T) {
	svc := &mockAddressService{}
	h := handler.NewAddressHandler(svc)

	mux := newMux("GET /addresses/{id}", h.GetAddress)
	req := httptest.NewRequest(http.MethodGet, "/addresses/abc", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestGetAddressHandler_NotFound(t *testing.T) {
	svc := &mockAddressService{
		getFn: func(id, userID int) (*model.Address, error) {
			return nil, service.ErrAddressNotFound
		},
	}
	h := handler.NewAddressHandler(svc)

	mux := newMux("GET /addresses/{id}", h.GetAddress)
	req := httptest.NewRequest(http.MethodGet, "/addresses/999", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("beklenen 404, alınan %d", rr.Code)
	}
}

func TestGetAddressHandler_AccessDenied(t *testing.T) {
	svc := &mockAddressService{
		getFn: func(id, userID int) (*model.Address, error) {
			return nil, service.ErrAccessDenied
		},
	}
	h := handler.NewAddressHandler(svc)

	mux := newMux("GET /addresses/{id}", h.GetAddress)
	req := httptest.NewRequest(http.MethodGet, "/addresses/1", nil)
	req.Header.Set("X-User-ID", "99")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}

// ── CreateAddress handler testleri ────────────────────────────────────────────

func TestCreateAddressHandler_Success(t *testing.T) {
	svc := &mockAddressService{
		createFn: func(userID int, req model.AddressRequest) (*model.Address, error) {
			return fakeAddress(), nil
		},
	}
	h := handler.NewAddressHandler(svc)

	body := toJSON(t, map[string]string{
		"title":         "Ev",
		"address_line1": "Test Sokak No:1",
		"city":          "İstanbul",
		"postal_code":   "34000",
	})
	req := httptest.NewRequest(http.MethodPost, "/addresses", body)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.CreateAddress(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("beklenen 201, alınan %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateAddressHandler_InvalidJSON(t *testing.T) {
	svc := &mockAddressService{}
	h := handler.NewAddressHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/addresses", bytes.NewBufferString("bad json"))
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.CreateAddress(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestCreateAddressHandler_ValidationError(t *testing.T) {
	svc := &mockAddressService{
		createFn: func(userID int, req model.AddressRequest) (*model.Address, error) {
			return nil, service.ErrTitleRequired
		},
	}
	h := handler.NewAddressHandler(svc)

	body := toJSON(t, map[string]string{"title": "", "address_line1": "X", "city": "Y", "postal_code": "Z"})
	req := httptest.NewRequest(http.MethodPost, "/addresses", body)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.CreateAddress(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

// ── UpdateAddress handler testleri ────────────────────────────────────────────

func TestUpdateAddressHandler_Success(t *testing.T) {
	svc := &mockAddressService{
		updateFn: func(id, userID int, req model.AddressRequest) (*model.Address, error) {
			a := fakeAddress()
			a.Title = req.Title
			return a, nil
		},
	}
	h := handler.NewAddressHandler(svc)

	body := toJSON(t, map[string]string{
		"title": "İş", "address_line1": "Yeni Adres", "city": "Ankara", "postal_code": "06000",
	})
	mux := newMux("PUT /addresses/{id}", h.UpdateAddress)
	req := httptest.NewRequest(http.MethodPut, "/addresses/1", body)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateAddressHandler_InvalidID(t *testing.T) {
	svc := &mockAddressService{}
	h := handler.NewAddressHandler(svc)

	mux := newMux("PUT /addresses/{id}", h.UpdateAddress)
	req := httptest.NewRequest(http.MethodPut, "/addresses/abc", bytes.NewBufferString("{}"))
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestUpdateAddressHandler_AccessDenied(t *testing.T) {
	svc := &mockAddressService{
		updateFn: func(id, userID int, req model.AddressRequest) (*model.Address, error) {
			return nil, service.ErrAccessDenied
		},
	}
	h := handler.NewAddressHandler(svc)

	body := toJSON(t, map[string]string{
		"title": "X", "address_line1": "X", "city": "X", "postal_code": "X",
	})
	mux := newMux("PUT /addresses/{id}", h.UpdateAddress)
	req := httptest.NewRequest(http.MethodPut, "/addresses/1", body)
	req.Header.Set("X-User-ID", "99")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}

// ── DeleteAddress handler testleri ────────────────────────────────────────────

func TestDeleteAddressHandler_Success(t *testing.T) {
	svc := &mockAddressService{
		deleteFn: func(id, userID int) error { return nil },
	}
	h := handler.NewAddressHandler(svc)

	mux := newMux("DELETE /addresses/{id}", h.DeleteAddress)
	req := httptest.NewRequest(http.MethodDelete, "/addresses/1", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestDeleteAddressHandler_InvalidID(t *testing.T) {
	svc := &mockAddressService{}
	h := handler.NewAddressHandler(svc)

	mux := newMux("DELETE /addresses/{id}", h.DeleteAddress)
	req := httptest.NewRequest(http.MethodDelete, "/addresses/abc", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestDeleteAddressHandler_AccessDenied(t *testing.T) {
	svc := &mockAddressService{
		deleteFn: func(id, userID int) error { return service.ErrAccessDenied },
	}
	h := handler.NewAddressHandler(svc)

	mux := newMux("DELETE /addresses/{id}", h.DeleteAddress)
	req := httptest.NewRequest(http.MethodDelete, "/addresses/1", nil)
	req.Header.Set("X-User-ID", "99")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}

func TestDeleteAddressHandler_NotFound(t *testing.T) {
	svc := &mockAddressService{
		deleteFn: func(id, userID int) error { return service.ErrAddressNotFound },
	}
	h := handler.NewAddressHandler(svc)

	mux := newMux("DELETE /addresses/{id}", h.DeleteAddress)
	req := httptest.NewRequest(http.MethodDelete, "/addresses/999", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("beklenen 404, alınan %d", rr.Code)
	}
}

// ── Health handler testi ──────────────────────────────────────────────────────

func TestHealthHandler_Success(t *testing.T) {
	svc := &mockAddressService{}
	h := handler.NewAddressHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}
