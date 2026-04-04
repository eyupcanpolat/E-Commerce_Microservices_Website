package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"eticaret/api-gateway/internal/middleware"
	sharedJWT "eticaret/shared/jwt"
)

// ── Yardımcı ─────────────────────────────────────────────────────────────────

func okHandler(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

func makeToken(t *testing.T, role string) string {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret-key")
	token, err := sharedJWT.GenerateToken(1, "test@example.com", role, "Test", "User")
	if err != nil {
		t.Fatalf("token üretilemedi: %v", err)
	}
	return token
}

// ── JWTAuth testleri ──────────────────────────────────────────────────────────

func TestJWTAuth_MissingHeader(t *testing.T) {
	h := middleware.JWTAuth(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("beklenen 401, alınan %d", rr.Code)
	}
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	h := middleware.JWTAuth(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "InvalidToken")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("beklenen 401, alınan %d", rr.Code)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	h := middleware.JWTAuth(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("beklenen 401, alınan %d", rr.Code)
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	h := middleware.JWTAuth(http.HandlerFunc(okHandler))

	token := makeToken(t, "customer")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestJWTAuth_InjectsUserHeaders(t *testing.T) {
	var capturedUserID, capturedRole, capturedEmail string

	h := middleware.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = r.Header.Get("X-User-ID")
		capturedRole = r.Header.Get("X-User-Role")
		capturedEmail = r.Header.Get("X-User-Email")
		w.WriteHeader(http.StatusOK)
	}))

	token := makeToken(t, "admin")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if capturedUserID == "" {
		t.Error("X-User-ID header inject edilmeli")
	}
	if capturedRole != "admin" {
		t.Errorf("beklenen role admin, alınan %s", capturedRole)
	}
	if capturedEmail != "test@example.com" {
		t.Errorf("beklenen email test@example.com, alınan %s", capturedEmail)
	}
}

// ── RequireRole testleri ──────────────────────────────────────────────────────

func TestRequireRole_CorrectRole(t *testing.T) {
	h := middleware.RequireRole("admin")(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Role", "admin")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestRequireRole_WrongRole(t *testing.T) {
	h := middleware.RequireRole("admin")(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Role", "customer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}

func TestRequireRole_MissingRole(t *testing.T) {
	h := middleware.RequireRole("admin")(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}

// ── InjectInternalSecret testleri ─────────────────────────────────────────────

func TestInjectInternalSecret_AddsHeader(t *testing.T) {
	var capturedSecret string

	h := middleware.InjectInternalSecret(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSecret = r.Header.Get(middleware.InternalSecretHeader)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if capturedSecret == "" {
		t.Error("X-Internal-Secret header inject edilmeli")
	}
}

// ── CORS testleri ─────────────────────────────────────────────────────────────

func TestCORS_SetsHeaders(t *testing.T) {
	h := middleware.CORS(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Access-Control-Allow-Origin: * olmalı")
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Access-Control-Allow-Methods header olmalı")
	}
}

func TestCORS_PreflightReturns204(t *testing.T) {
	h := middleware.CORS(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("OPTIONS için beklenen 204, alınan %d", rr.Code)
	}
}

// ── Chain testleri ────────────────────────────────────────────────────────────

func TestChain_AppliesInOrder(t *testing.T) {
	order := []string{}

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1")
			next.ServeHTTP(w, r)
		})
	}
	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2")
			next.ServeHTTP(w, r)
		})
	}

	h := middleware.Chain(http.HandlerFunc(okHandler), m1, m2)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if len(order) != 2 || order[0] != "m1" || order[1] != "m2" {
		t.Errorf("middleware sırası yanlış: %v", order)
	}
}
