// main.go — CDN / Static File Service
//
// Bu servis ürün görsellerini serve eder.
// Product Service sadece image path'i tutar (örn: /images/laptop.jpg)
// Gerçek dosya bu servisten alınır.
//
// NETWORK ISOLATION: X-Internal-Secret zorunlu değildir (public CDN),
// ancak isteğe bağlı olarak aktif edilebilir.
// Statik dosyalar: /images/* path'i altında serve edilir.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"eticaret/shared/logger"
)

func main() {
	// STATIC_DIR env var veya varsayılan /app/static (Docker için)
	baseDir := os.Getenv("STATIC_DIR")
	if baseDir == "" {
		baseDir = "/app/static"
	}
	// Yerel geliştirme: static klasörü yoksa, çalışma dizininin yanında ara
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		if wd, wdErr := os.Getwd(); wdErr == nil {
			local := filepath.Join(wd, "static")
			if _, lErr := os.Stat(local); lErr == nil {
				baseDir = local
			}
		}
	}

	port := os.Getenv("CDN_SERVICE_PORT")
	if port == "" {
		port = "8085"
	}

	mux := http.NewServeMux()

	// ── Health endpoint ────────────────────────────────────────────────────
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"service": "cdn-service",
			"status":  "ok",
			"base_dir": baseDir,
		})
	})

	// ── Static files: /images/* ────────────────────────────────────────────
	// Tüm /images/ istekleri staticDir/images/ altındaki dosyalara yönlendirilir.
	imagesDir := filepath.Join(baseDir, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		logger.Fatal("Could not create images directory", "path", imagesDir, "error", err)
	}

	// FileServer with CORS headers (images are accessed directly by browser)
	imageServer := http.FileServer(http.Dir(imagesDir))
	// Önemli: method prefix kullan, GET / ile çakışmayı önle
	mux.Handle("GET /images/", addCORSAndCache(http.StripPrefix("/images/", imageServer)))
	mux.Handle("OPTIONS /images/", addCORSAndCache(http.StripPrefix("/images/", imageServer)))

	// ── Default no-image placeholder (SVG) ────────────────────────────────
	mux.HandleFunc("/placeholder.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(placeholderSVG))
	})

	// ── Root: list available images (dev only) ─────────────────────────────
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		files := listFiles(imagesDir)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service":     "cdn-service",
			"images_path": "/images/",
			"file_count":  len(files),
			"files":       files,
		})
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("CDN Service starting",
		"port", port,
		"static_dir", baseDir,
		"images_url", fmt.Sprintf("http://localhost:%s/images/", port),
	)

	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("CDN Service failed", "error", err)
	}
}

// addCORSAndCache adds appropriate headers for static asset serving.
func addCORSAndCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour cache
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// listFiles lists filenames in the given directory (non-recursive, for dev listing).
func listFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// placeholderSVG is returned for /placeholder.svg — used when a product has no image.
const placeholderSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="400" viewBox="0 0 400 400">
  <rect width="400" height="400" fill="#161a23"/>
  <rect x="1" y="1" width="398" height="398" fill="none" stroke="rgba(108,99,255,0.3)" stroke-width="1"/>
  <g opacity="0.4">
    <rect x="140" y="120" width="120" height="90" rx="8" fill="none" stroke="#6c63ff" stroke-width="2"/>
    <circle cx="165" cy="145" r="12" fill="#6c63ff" opacity="0.5"/>
    <polyline points="140,210 175,165 205,185 235,150 260,210" fill="none" stroke="#6c63ff" stroke-width="2"/>
  </g>
  <text x="200" y="255" text-anchor="middle" font-family="Inter,sans-serif" font-size="13" fill="#4a5068">Görsel Yok</text>
</svg>`
