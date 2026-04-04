package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheckHandler(t *testing.T) {
	// İstek oluşturuluyor
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Yanıtı kaydetmek için bir recorder oluşturuluyor
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheckHandler)

	handler.ServeHTTP(rr, req)

	// Durum kodu kontrolü
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler yanlış durum kodu döndürdü: beklenen %v alınan %v",
			http.StatusOK, status)
	}

	// Body kontrolü
	expected := `{"status": "ok"}`
	if rr.Body.String() != expected {
		t.Errorf("Handler beklenmeyen bir body döndürdü: beklenen %v alınan %v",
			expected, rr.Body.String())
	}
}
