package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"eticaret/order-service/internal/middleware"
)

func okHandler(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

func TestNetworkIsolation_MissingSecret(t *testing.T) {
	h := middleware.NetworkIsolation(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}

func TestNetworkIsolation_WrongSecret(t *testing.T) {
	h := middleware.NetworkIsolation(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Internal-Secret", "yanlis-secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}

func TestNetworkIsolation_CorrectSecret(t *testing.T) {
	os.Setenv("INTERNAL_SECRET", "test-internal-secret")
	defer os.Unsetenv("INTERNAL_SECRET")
	h := middleware.NetworkIsolation(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Internal-Secret", "test-internal-secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestGetUserID_WithHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "42")
	if middleware.GetUserID(req) != 42 {
		t.Error("beklenen 42")
	}
}

func TestGetUserID_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if middleware.GetUserID(req) != 0 {
		t.Error("beklenen 0")
	}
}

func TestGetUserID_InvalidValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "abc")
	if middleware.GetUserID(req) != 0 {
		t.Error("geçersiz değer için beklenen 0")
	}
}

func TestGetUserRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Role", "admin")
	if middleware.GetUserRole(req) != "admin" {
		t.Error("beklenen role admin")
	}
}

func TestRequireUser_WithUserID(t *testing.T) {
	h := middleware.RequireUser(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestRequireUser_WithoutUserID(t *testing.T) {
	h := middleware.RequireUser(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("beklenen 401, alınan %d", rr.Code)
	}
}

func TestRequireAdmin_WithAdminRole(t *testing.T) {
	h := middleware.RequireAdmin(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Role", "admin")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestRequireAdmin_WithCustomerRole(t *testing.T) {
	h := middleware.RequireAdmin(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Role", "customer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}
