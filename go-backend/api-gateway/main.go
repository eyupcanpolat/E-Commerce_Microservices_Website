package main

import (
	"fmt"
	"log"
	"net/http"
)

// Testi geçmek için gereken işleyici fonksiyon
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status": "ok"}`)
}

func main() {
	http.HandleFunc("/health", HealthCheckHandler)

	log.Println("API Gateway 8080 portunda çalışıyor...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
