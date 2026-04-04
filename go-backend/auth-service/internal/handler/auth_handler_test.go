package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"eticaret/auth-service/internal/handler"
	"eticaret/auth-service/internal/model"
	"eticaret/auth-service/internal/service"
)

// ── Mock AuthService ──────────────────────────────────────────────────────────

type mockAuthService struct {
	registerFn func(req model.RegisterRequest) (*model.AuthResponse, error)
	loginFn    func(req model.LoginRequest) (*model.AuthResponse, error)
	updateFn   func(userID int, req model.UpdateProfileRequest) (*model.User, error)
}

func (m *mockAuthService) Register(req model.RegisterRequest) (*model.AuthResponse, error) {
	return m.registerFn(req)
}

func (m *mockAuthService) Login(req model.LoginRequest) (*model.AuthResponse, error) {
	return m.loginFn(req)
}

func (m *mockAuthService) UpdateProfile(userID int, req model.UpdateProfileRequest) (*model.User, error) {
	return m.updateFn(userID, req)
}

// ── Yardımcı ─────────────────────────────────────────────────────────────────

func toJSON(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("JSON marshal hatası: %v", err)
	}
	return bytes.NewBuffer(b)
}

func fakeAuthResponse() *model.AuthResponse {
	r := &model.AuthResponse{Token: "fake-jwt-token", ExpiresIn: 86400}
	r.User.ID = 1
	r.User.Email = "test@example.com"
	r.User.Role = "customer"
	r.User.FirstName = "Test"
	r.User.LastName = "User"
	return r
}

// ── Register handler testleri ─────────────────────────────────────────────────

func TestRegisterHandler_Success(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(req model.RegisterRequest) (*model.AuthResponse, error) {
			return fakeAuthResponse(), nil
		},
	}
	h := handler.NewAuthHandler(svc)

	body := toJSON(t, map[string]string{
		"email":            "test@example.com",
		"password":         "password123",
		"password_confirm": "password123",
		"first_name":       "Test",
		"last_name":        "User",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("beklenen 201, alınan %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestRegisterHandler_InvalidJSON(t *testing.T) {
	svc := &mockAuthService{}
	h := handler.NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestRegisterHandler_EmailExists(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(req model.RegisterRequest) (*model.AuthResponse, error) {
			return nil, service.ErrEmailExists
		},
	}
	h := handler.NewAuthHandler(svc)

	body := toJSON(t, map[string]string{
		"email": "existing@example.com", "password": "password123",
		"password_confirm": "password123", "first_name": "Test", "last_name": "User",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("beklenen 409, alınan %d", rr.Code)
	}
}

func TestRegisterHandler_ValidationError(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(req model.RegisterRequest) (*model.AuthResponse, error) {
			return nil, service.ErrPasswordTooShort
		},
	}
	h := handler.NewAuthHandler(svc)

	body := toJSON(t, map[string]string{
		"email": "test@example.com", "password": "abc",
		"password_confirm": "abc", "first_name": "Test", "last_name": "User",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

// ── Login handler testleri ────────────────────────────────────────────────────

func TestLoginHandler_Success(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(req model.LoginRequest) (*model.AuthResponse, error) {
			return fakeAuthResponse(), nil
		},
	}
	h := handler.NewAuthHandler(svc)

	body := toJSON(t, map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	data, _ := resp["data"].(map[string]interface{})
	if data["token"] == "" {
		t.Error("response'da token olmalı")
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(req model.LoginRequest) (*model.AuthResponse, error) {
			return nil, service.ErrInvalidCredentials
		},
	}
	h := handler.NewAuthHandler(svc)

	body := toJSON(t, map[string]string{
		"email": "test@example.com", "password": "wrong",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("beklenen 401, alınan %d", rr.Code)
	}
}

func TestLoginHandler_InvalidJSON(t *testing.T) {
	svc := &mockAuthService{}
	h := handler.NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("{bad json"))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

// ── UpdateProfile handler testleri ───────────────────────────────────────────

func TestUpdateProfileHandler_NoUserID(t *testing.T) {
	svc := &mockAuthService{}
	h := handler.NewAuthHandler(svc)

	// X-User-ID header yok → 401 olmalı
	req := httptest.NewRequest(http.MethodPut, "/auth/profile", toJSON(t, map[string]string{
		"first_name": "Yeni",
	}))
	rr := httptest.NewRecorder()

	h.UpdateProfile(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("beklenen 401, alınan %d", rr.Code)
	}
}

func TestUpdateProfileHandler_Success(t *testing.T) {
	svc := &mockAuthService{
		updateFn: func(userID int, req model.UpdateProfileRequest) (*model.User, error) {
			return &model.User{
				ID:        userID,
				Email:     "test@example.com",
				FirstName: req.FirstName,
				LastName:  "User",
			}, nil
		},
	}
	h := handler.NewAuthHandler(svc)

	body := toJSON(t, map[string]string{"first_name": "Yeni"})
	req := httptest.NewRequest(http.MethodPut, "/auth/profile", body)
	req.Header.Set("X-User-ID", "1") // gateway'in inject ettiği header
	rr := httptest.NewRecorder()

	h.UpdateProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateProfileHandler_InvalidJSON(t *testing.T) {
	svc := &mockAuthService{}
	h := handler.NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPut, "/auth/profile", bytes.NewBufferString("bad json"))
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()

	h.UpdateProfile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

// ── Health handler testleri ───────────────────────────────────────────────────

func TestHealthHandler_Unauthenticated(t *testing.T) {
	svc := &mockAuthService{}
	h := handler.NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	h.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestHealthHandler_Authenticated(t *testing.T) {
	svc := &mockAuthService{}
	h := handler.NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-User-ID", "42")
	rr := httptest.NewRecorder()

	h.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}
