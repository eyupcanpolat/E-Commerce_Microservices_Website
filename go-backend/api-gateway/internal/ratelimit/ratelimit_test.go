package ratelimit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"eticaret/api-gateway/internal/ratelimit"
)

// ── Allow testleri ────────────────────────────────────────────────────────────

func TestAllow_FirstRequest(t *testing.T) {
	l := ratelimit.NewLimiter(60)

	if !l.Allow("192.168.1.1") {
		t.Error("ilk istek izin verilmeli")
	}
}

func TestAllow_BurstLimit(t *testing.T) {
	// 60 req/min → burst = 60/4 = 15
	l := ratelimit.NewLimiter(60)

	ip := "10.0.0.1"
	allowed := 0
	for i := 0; i < 20; i++ {
		if l.Allow(ip) {
			allowed++
		}
	}

	// Burst kapasitesi (15) aşılmalı, 20 istek hepsi geçmemeli
	if allowed >= 20 {
		t.Errorf("burst limit aşılmalıydı, %d/20 geçti", allowed)
	}
	if allowed == 0 {
		t.Error("en az 1 istek geçmeli")
	}
}

func TestAllow_DifferentIPs(t *testing.T) {
	// 4 req/min → burst = 1
	l := ratelimit.NewLimiter(4)

	// IP1 limitini tüket
	ip1 := "1.1.1.1"
	for l.Allow(ip1) {
	}

	// IP2 hâlâ geçebilmeli
	if !l.Allow("2.2.2.2") {
		t.Error("farklı IP için rate limit bağımsız olmalı")
	}
}

// ── Middleware testleri ───────────────────────────────────────────────────────

func TestMiddleware_AllowsNormalRequests(t *testing.T) {
	l := ratelimit.NewLimiter(60)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:1234"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestMiddleware_BlocksAfterBurst(t *testing.T) {
	// 4 req/min → burst = 1, hızlıca limitlenir
	l := ratelimit.NewLimiter(4)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "10.0.0.5"
	blocked := false
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":9999"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			blocked = true
			break
		}
	}

	if !blocked {
		t.Error("burst aşıldığında 429 dönmeli")
	}
}

func TestMiddleware_RetryAfterHeader(t *testing.T) {
	l := ratelimit.NewLimiter(4)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "10.0.0.6"
	var lastRR *httptest.ResponseRecorder
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":9999"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		lastRR = rr
		if rr.Code == http.StatusTooManyRequests {
			break
		}
	}

	if lastRR != nil && lastRR.Code == http.StatusTooManyRequests {
		if lastRR.Header().Get("Retry-After") == "" {
			t.Error("429 yanıtında Retry-After header olmalı")
		}
	}
}

func TestMiddleware_XForwardedFor(t *testing.T) {
	l := ratelimit.NewLimiter(60)
	var capturedPass bool
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPass = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if !capturedPass {
		t.Error("X-Forwarded-For ile gelen istek geçmeli")
	}
}
