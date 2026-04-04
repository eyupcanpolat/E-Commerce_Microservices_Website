package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"eticaret/auth-service/internal/middleware"
)

func okHandler(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

// ── NetworkIsolation testleri ─────────────────────────────────────────────────

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

// ── GetUserID testleri ────────────────────────────────────────────────────────

func TestGetUserID_WithHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "42")

	id := middleware.GetUserID(req)
	if id != 42 {
		t.Errorf("beklenen 42, alınan %d", id)
	}
}

func TestGetUserID_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	id := middleware.GetUserID(req)
	if id != 0 {
		t.Errorf("beklenen 0, alınan %d", id)
	}
}

func TestGetUserID_InvalidValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "abc")

	id := middleware.GetUserID(req)
	if id != 0 {
		t.Errorf("geçersiz değer için beklenen 0, alınan %d", id)
	}
}

// ── GetUserRole / GetUserEmail testleri ───────────────────────────────────────

func TestGetUserRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Role", "admin")

	if middleware.GetUserRole(req) != "admin" {
		t.Error("beklenen role admin")
	}
}

func TestGetUserEmail(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Email", "test@example.com")

	if middleware.GetUserEmail(req) != "test@example.com" {
		t.Error("beklenen email test@example.com")
	}
}

// ── RequireUser testleri ──────────────────────────────────────────────────────

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
