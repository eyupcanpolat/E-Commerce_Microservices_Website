package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"eticaret/api-gateway/internal/handler"
	sharedJWT "eticaret/shared/jwt"
)

// ── Yardımcı fonksiyonlar ─────────────────────────────────────────────────────

// fakeBackend istekleri kaydeden sahte bir backend servisi oluşturur.
func fakeBackend(t *testing.T) (*httptest.Server, *http.Request) {
	t.Helper()
	var captured *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, captured
}

// makeToken test için geçerli bir JWT token üretir.
func makeToken(t *testing.T, role string) string {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret-key")
	token, err := sharedJWT.GenerateToken(1, "test@example.com", role, "Test", "User")
	if err != nil {
		t.Fatalf("token üretilemedi: %v", err)
	}
	return token
}

// ── HealthHandler testleri ────────────────────────────────────────────────────

func TestHealthHandler_StatusOK(t *testing.T) {
	urls := map[string]string{
		"/auth":      "http://localhost:8081",
		"/products":  "http://localhost:8082",
		"/addresses": "http://localhost:8083",
		"/orders":    "http://localhost:8084",
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.HealthHandler(urls).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestHealthHandler_ResponseBody(t *testing.T) {
	urls := map[string]string{"/auth": "http://localhost:8081"}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.HealthHandler(urls).ServeHTTP(rr, req)

	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("JSON parse hatası: %v", err)
	}

	if body["success"] != true {
		t.Errorf("success alanı true olmalı, alınan: %v", body["success"])
	}
	if body["service"] != "api-gateway" {
		t.Errorf("service alanı 'api-gateway' olmalı, alınan: %v", body["service"])
	}
	if body["status"] != "ok" {
		t.Errorf("status alanı 'ok' olmalı, alınan: %v", body["status"])
	}
}

func TestHealthHandler_ContentTypeJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.HealthHandler(nil).ServeHTTP(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type 'application/json' olmalı, alınan: %s", ct)
	}
}

// ── NewAuthHandler testleri ───────────────────────────────────────────────────

func TestAuthHandler_LoginIsPublic(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewAuthHandler(proxy)

	// Token olmadan POST /auth/login → 200 olmalı (public)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("login public olmalı, beklenen 200, alınan %d", rr.Code)
	}
}

func TestAuthHandler_RegisterIsPublic(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewAuthHandler(proxy)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("register public olmalı, beklenen 200, alınan %d", rr.Code)
	}
}

func TestAuthHandler_ProfileRequiresJWT(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewAuthHandler(proxy)

	// Token olmadan PUT /auth/profile → 401 olmalı
	req := httptest.NewRequest(http.MethodPut, "/auth/profile", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("profile JWT gerektirmeli, beklenen 401, alınan %d", rr.Code)
	}
}

func TestAuthHandler_ProfileWithValidJWT(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewAuthHandler(proxy)

	token := makeToken(t, "customer")
	req := httptest.NewRequest(http.MethodPut, "/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("geçerli token ile profile erişilebilmeli, beklenen 200, alınan %d", rr.Code)
	}
}

// ── NewProductHandler testleri ────────────────────────────────────────────────

func TestProductHandler_GETIsPublic(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewProductHandler(proxy)

	// Token olmadan GET /products → 200 olmalı
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET products public olmalı, beklenen 200, alınan %d", rr.Code)
	}
}

func TestProductHandler_POSTRequiresJWT(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewProductHandler(proxy)

	// Token olmadan POST /products → 401 olmalı
	req := httptest.NewRequest(http.MethodPost, "/products", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("POST products JWT gerektirmeli, beklenen 401, alınan %d", rr.Code)
	}
}

func TestProductHandler_POSTRequiresAdminRole(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewProductHandler(proxy)

	// customer rolüyle POST /products → 403 olmalı
	token := makeToken(t, "customer")
	req := httptest.NewRequest(http.MethodPost, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("customer POST yapamamalı, beklenen 403, alınan %d", rr.Code)
	}
}

func TestProductHandler_POSTWithAdminRole(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewProductHandler(proxy)

	// admin rolüyle POST /products → 200 olmalı
	token := makeToken(t, "admin")
	req := httptest.NewRequest(http.MethodPost, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("admin POST yapabilmeli, beklenen 200, alınan %d", rr.Code)
	}
}

func TestProductHandler_DELETERequiresAdmin(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewProductHandler(proxy)

	token := makeToken(t, "customer")
	req := httptest.NewRequest(http.MethodDelete, "/products/123", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("customer DELETE yapamamalı, beklenen 403, alınan %d", rr.Code)
	}
}

// ── NewOrderHandler testleri ──────────────────────────────────────────────────

func TestOrderHandler_RequiresJWT(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewOrderHandler(proxy)

	// Token olmadan GET /orders → 401 olmalı
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("orders JWT gerektirmeli, beklenen 401, alınan %d", rr.Code)
	}
}

func TestOrderHandler_CustomerCanListOrders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewOrderHandler(proxy)

	token := makeToken(t, "customer")
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("customer siparişlerini görebilmeli, beklenen 200, alınan %d", rr.Code)
	}
}

func TestOrderHandler_StatusUpdateRequiresAdmin(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewOrderHandler(proxy)

	// customer rolüyle PUT /orders/1/status → 403 olmalı
	token := makeToken(t, "customer")
	req := httptest.NewRequest(http.MethodPut, "/orders/1/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("customer status güncelleyememeli, beklenen 403, alınan %d", rr.Code)
	}
}

func TestOrderHandler_AdminCanUpdateStatus(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL)
	h := handler.NewOrderHandler(proxy)

	// admin rolüyle PUT /orders/1/status → 200 olmalı
	token := makeToken(t, "admin")
	req := httptest.NewRequest(http.MethodPut, "/orders/1/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("admin status güncelleyebilmeli, beklenen 200, alınan %d", rr.Code)
	}
}

// ── Test yardımcısı ───────────────────────────────────────────────────────────

// newTestProxy gerçek bir httptest.Server'a yönlendiren proxy döner.
func newTestProxy(targetURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := http.NewRequest(r.Method, targetURL+r.URL.Path, r.Body)
		for k, v := range r.Header {
			req.Header[k] = v
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "proxy error", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
	})
}
