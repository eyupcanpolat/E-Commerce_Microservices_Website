# E-Ticaret Go Mikroservis Backend

PHP'den Go'ya tam dönüşüm — mikroservis mimarisi.

---

## 🗂️ Proje Klasör Yapısı

```
go-backend/
├── api-gateway/                  # Port 8080 — Tüm trafiği yönlendirir
│   ├── cmd/main.go               # Giriş noktası
│   ├── internal/
│   │   ├── middleware/           # CORS, logger, opsiyonel JWT
│   │   └── proxy/                # Reverse proxy router
│   ├── Dockerfile
│   └── go.mod
│
├── auth-service/                 # Port 8081 — JWT üretimi
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── handler/              # HTTP handler'ları
│   │   ├── middleware/           # JWT doğrulama + context helper
│   │   ├── model/                # User, LoginRequest, AuthResponse
│   │   ├── repository/           # JSON dosya erişimi
│   │   └── service/              # İş mantığı + bcrypt
│   ├── data/users.json           # Mock kullanıcı verisi
│   ├── tests/                    # Unit testler
│   ├── Dockerfile
│   └── go.mod
│
├── product-service/              # Port 8082
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── handler/
│   │   ├── middleware/           # JWT + admin rol kontrolü
│   │   ├── model/                # Product, Category, Filter
│   │   ├── repository/           # Filtreli JSON erişimi
│   │   └── service/
│   ├── data/
│   │   ├── products.json
│   │   └── categories.json
│   ├── tests/
│   ├── Dockerfile
│   └── go.mod
│
├── address-service/              # Port 8083
│   ├── cmd/main.go
│   ├── internal/                 # Handler + MW + Model + Repo + Service
│   ├── data/addresses.json
│   ├── Dockerfile
│   └── go.mod
│
├── order-service/                # Port 8084
│   ├── cmd/main.go
│   ├── internal/                 # Handler + MW + Model + Repo + Service
│   ├── data/orders.json
│   ├── Dockerfile
│   └── go.mod
│
├── shared/                       # Tüm servisler tarafından kullanılan ortak kod
│   ├── jwt/jwt.go                # Token üretimi ve doğrulaması
│   ├── response/response.go      # Standart JSON yanıt yardımcıları
│   ├── logger/logger.go          # Yapılandırılmış JSON logger
│   └── go.mod
│
├── docker-compose.yml            # Tüm sistemi tek komutla başlatır
├── .env.example                  # Environment değişkenleri şablonu
└── .gitignore
```

---

## 🏛️ Mimari

```
Frontend (PHP Views — değişmedi)
        │
        ▼
  ┌─────────────┐
  │ API Gateway  │  :8080  (tek genel giriş noktası)
  │  - CORS      │
  │  - Logging   │
  └──────┬───────┘
         │ HTTP reverse proxy
    ┌────┴────────────────────────────┐
    │         │           │           │
    ▼         ▼           ▼           ▼
┌───────┐ ┌────────┐ ┌────────┐ ┌────────┐
│ Auth  │ │Product │ │Address │ │ Order  │
│ :8081 │ │ :8082  │ │ :8083  │ │ :8084  │
└───────┘ └────────┘ └────────┘ └───┬────┘
    │                                │
    │   JWT_SECRET (ortak env)       │ HTTP GET /products/{id}
    └────────────────────────────────┘
         Servisler arası iletişim
         (Order → Product)
```

---

## 🔐 JWT Akışı

1. **Login**: `POST /auth/login` → AuthService bcrypt ile şifreyi doğrular → JWT üretir → döner
2. **Korumalı istek**: Client `Authorization: Bearer <token>` header'ı ekler
3. **Doğrulama**: Her servis kendi JWT middleware'i ile `shared/jwt.ValidateToken()` çağırır
4. **Forbidden**: Admin gerektiren endpoint'lere customer rolüyle erişilirse 403 döner

### Token İçeriği
```json
{
  "userId": 2,
  "email": "ahmet@example.com",
  "role": "customer",
  "firstName": "Ahmet",
  "lastName": "Yılmaz",
  "exp": 1712345678,
  "iat": 1712259278,
  "iss": "eticaret-auth-service"
}
```

---

## 🚀 Çalıştırma

### Docker ile (Önerilen)
```bash
cd go-backend
cp .env.example .env
docker-compose up --build
```

### Yerel Geliştirme (Go yüklü olmalı)
```bash
# Her servis için ayrı terminalde:
cd go-backend/shared && go mod download

cd ../auth-service
go mod download
JWT_SECRET=my-secret go run ./cmd

cd ../product-service
go mod download
JWT_SECRET=my-secret go run ./cmd

cd ../address-service
go mod download
JWT_SECRET=my-secret go run ./cmd

cd ../order-service
go mod download
JWT_SECRET=my-secret PRODUCT_SERVICE_URL=http://localhost:8082 go run ./cmd

cd ../api-gateway
go mod download
go run ./cmd
```

### Testleri Çalıştırma
```bash
cd go-backend/auth-service
go test ./tests/ -v

cd ../product-service
go test ./tests/ -v
```

---

## 📡 API Endpoint'leri

### Auth Service (:8081)
| Method | URL | Auth | Açıklama |
|--------|-----|------|----------|
| POST | /auth/register | ❌ | Kayıt |
| POST | /auth/login | ❌ | Giriş → JWT |
| GET | /health | ❌ | Sağlık kontrolü |

### Product Service (:8082)
| Method | URL | Auth | Açıklama |
|--------|-----|------|----------|
| GET | /products | ❌ | Liste (filtreleme) |
| GET | /products/{id} | ❌ | Ürün detayı |
| GET | /products/slug/{slug} | ❌ | Slug ile detay |
| GET | /products/featured | ❌ | Öne çıkan ürünler |
| GET | /products/search?q= | ❌ | Arama |
| POST | /products | ✅ Admin | Yeni ürün |
| DELETE | /products/{id} | ✅ Admin | Ürün sil |

### Address Service (:8083)
| Method | URL | Auth | Açıklama |
|--------|-----|------|----------|
| GET | /addresses | ✅ | Adreslerim |
| POST | /addresses | ✅ | Yeni adres |
| GET | /addresses/{id} | ✅ | Adres detayı |
| PUT | /addresses/{id} | ✅ | Adres güncelle |
| DELETE | /addresses/{id} | ✅ | Adres sil |

### Order Service (:8084)
| Method | URL | Auth | Açıklama |
|--------|-----|------|----------|
| GET | /orders | ✅ | Siparişlerim |
| POST | /orders | ✅ | Sipariş oluştur |
| GET | /orders/{id} | ✅ | Sipariş detayı |
| GET | /orders/number/{no} | ✅ | Sipariş no ile detay |
| POST | /orders/{no}/cancel | ✅ | İptal et |
| PUT | /orders/{id}/status | ✅ Admin | Durum güncelle |

---

## 📬 Örnek HTTP İstekleri

### 1. Kayıt
```http
POST http://localhost:8080/auth/register
Content-Type: application/json

{
  "email": "yeni@kullanici.com",
  "password": "guclusifre123",
  "password_confirm": "guclusifre123",
  "first_name": "Ali",
  "last_name": "Veli",
  "phone": "+90 555 000 0001"
}
```

**Yanıt:**
```json
{
  "success": true,
  "message": "Kayıt başarılı",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5...",
    "expires_in": 86400,
    "user": { "id": 4, "email": "yeni@kullanici.com", "role": "customer" }
  }
}
```

### 2. Ürün Listeleme (Filtreli)
```http
GET http://localhost:8080/products?category=4&min_price=1000&sort=price_asc&page=1
```

### 3. Sipariş Oluşturma
```http
POST http://localhost:8080/orders
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5...
Content-Type: application/json

{
  "shipping_address_id": 1,
  "shipping_method": "Hızlı Kargo",
  "payment_method": "credit_card",
  "items": [
    { "product_id": 1, "quantity": 1 }
  ]
}
```

---

## 🔄 PHP'den Go'ya Dönüşüm Özeti

| PHP Bileşeni | Go Karşılığı |
|---|---|
| `AuthController` | `auth-service/handler + service` |
| `ProductController` | `product-service/handler + service` |
| `OrderController` | `order-service/handler + service` |
| `UserController` (adresler) | `address-service` |
| `Session::isLoggedIn()` | JWT middleware |
| MySQL | JSON dosyaları (mock) |
| PHP Router | API Gateway (reverse proxy) |
| `password_hash()` | `bcrypt.GenerateFromPassword()` |

### Düzeltilen Güvenlik Açıkları (CTF'den)
- ❌ SQL Injection (`findByEmailVulnerable`) → ✅ Tip güvenli JSON sorguları
- ❌ IDOR (adres sahipliği kontrolü yok) → ✅ Her istek `userID` kontrolü yapıyor
- ❌ SSRF (`file_get_contents($avatarUrl)`) → ✅ Endpoint kaldırıldı
- ❌ XSS (arama sorgusu kaçışsız) → ✅ JSON API, HTML render etmiyor
