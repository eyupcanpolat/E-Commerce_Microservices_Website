# E-Ticaret Go Mikroservis Backend

PHP monolitik uygulamasından Go mikroservis mimarisine tam dönüşüm.

---

## Mimari Genel Bakış

```mermaid
graph TB
    Client["🌐 İstemci (Browser / Postman)"]

    subgraph Public["Dış Ağ (Host)"]
        GW["API Gateway\n:8080"]
        CDN["CDN Service\n:8085"]
    end

    subgraph Internal["Docker İç Ağ (eticaret-internal) — Dışarıdan Erişilemez"]
        AUTH["Auth Service\n:8081"]
        PRODUCT["Product Service\n:8082"]
        ADDRESS["Address Service\n:8083"]
        ORDER["Order Service\n:8084"]
        MONGO[("MongoDB\n:27017")]
    end

    Client -->|"HTTP :8080"| GW
    Client -->|"Statik dosyalar :8085"| CDN

    GW -->|"X-Internal-Secret ✓"| AUTH
    GW -->|"X-Internal-Secret ✓"| PRODUCT
    GW -->|"X-Internal-Secret ✓"| ADDRESS
    GW -->|"X-Internal-Secret ✓"| ORDER

    ORDER -->|"Stok sorgulama"| PRODUCT

    AUTH --- MONGO
    PRODUCT --- MONGO
    ADDRESS --- MONGO
    ORDER --- MONGO
    GW --- MONGO

    style Public fill:#e8f5e9,stroke:#388e3c
    style Internal fill:#fce4ec,stroke:#c62828
```

---

## Ağ İzolasyonu

```mermaid
sequenceDiagram
    participant C as İstemci
    participant GW as API Gateway :8080
    participant MS as Mikroservis (Auth/Product/...)

    Note over C,MS: ✅ NORMAL AKIŞ

    C->>GW: POST /auth/login (şifresiz header)
    GW->>GW: JWT doğrulama (gerekiyorsa)
    GW->>GW: X-Internal-Secret header ekle
    GW->>MS: POST /auth/login + X-Internal-Secret
    MS->>MS: Header kontrolü — SECRET doğru ✓
    MS->>GW: 200 OK + JWT Token
    GW->>C: 200 OK + JWT Token

    Note over C,MS: ❌ DOĞRUDAN ERİŞİM GİRİŞİMİ

    C-->>MS: POST /auth/login (X-Internal-Secret YOK)
    MS-->>C: 403 Forbidden (gateway'den gelmiyor)
```

---

## JWT Kimlik Doğrulama Akışı

```mermaid
sequenceDiagram
    participant C as İstemci
    participant GW as API Gateway
    participant AUTH as Auth Service
    participant SVC as Diğer Servis

    C->>GW: POST /auth/login {email, password}
    GW->>AUTH: Yönlendir (X-Internal-Secret ✓)
    AUTH->>AUTH: bcrypt şifre doğrulama
    AUTH->>C: JWT Token (24 saat geçerli)

    Note over C,SVC: Korumalı endpoint erişimi

    C->>GW: GET /orders (Authorization: Bearer TOKEN)
    GW->>GW: JWT doğrula — X-User-ID, X-User-Role header ekle
    GW->>SVC: GET /orders + X-User-* headers
    SVC->>SVC: Header'dan kullanıcıyı oku (JWT tekrar doğrulamaz)
    SVC->>C: 200 OK sipariş listesi
```

---

## Servis Haritası

| Servis | Port | Açıklama | Dış Erişim |
|--------|------|----------|------------|
| **API Gateway** | 8080 | Tek giriş noktası, JWT doğrulama, rate limit | ✅ |
| **Auth Service** | 8081 | Kayıt, giriş, profil | ❌ İç ağ |
| **Product Service** | 8082 | Ürün CRUD, arama, filtreleme | ❌ İç ağ |
| **Address Service** | 8083 | Kullanıcı adresleri | ❌ İç ağ |
| **Order Service** | 8084 | Sipariş yönetimi | ❌ İç ağ |
| **CDN Service** | 8085 | Statik dosya sunucu | ✅ |

---

## TDD — Test Sonuçları

```mermaid
graph LR
    subgraph GW["API Gateway (11 test)"]
        GW1["✅ HealthHandler"]
        GW2["✅ AuthHandler — public/JWT routing"]
        GW3["✅ ProductHandler — public/admin routing"]
        GW4["✅ OrderHandler — JWT/admin routing"]
        GW5["✅ JWTAuth middleware"]
        GW6["✅ RequireRole middleware"]
        GW7["✅ CORS middleware"]
        GW8["✅ Rate Limiter — burst/429"]
    end

    subgraph AUTH["Auth Service (12 test)"]
        A1["✅ Register — başarılı/duplicate/boş"]
        A2["✅ Login — başarılı/yanlış şifre/pasif hesap"]
        A3["✅ NetworkIsolation — secret kontrolü"]
        A4["✅ GetUserID / GetUserRole / GetUserEmail"]
    end

    subgraph PRODUCT["Product Service (14 test)"]
        P1["✅ ListProducts — filtreleme/kategori"]
        P2["✅ GetProduct — id/slug/bulunamadı"]
        P3["✅ CreateProduct — admin gerekli"]
        P4["✅ DeleteProduct — admin gerekli"]
        P5["✅ StockStatus — InStock/OutOfStock"]
        P6["✅ RequireAdmin middleware"]
    end

    subgraph ORDER["Order Service (15 test)"]
        O1["✅ CreateOrder — başarılı/stok yok/geçersiz"]
        O2["✅ ListOrders — JWT zorunlu"]
        O3["✅ GetOrder — sahiplik kontrolü"]
        O4["✅ CancelOrder — aktif sipariş"]
        O5["✅ UpdateStatus — admin gerekli"]
    end

    subgraph ADDRESS["Address Service (17 test)"]
        AD1["✅ ListAddresses — kullanıcıya göre"]
        AD2["✅ CreateAddress — doğrulama"]
        AD3["✅ UpdateAddress — sahiplik kontrolü"]
        AD4["✅ DeleteAddress — sahiplik kontrolü"]
        AD5["✅ RequireUser middleware"]
    end
```

### Test Çalıştırma

```bash
# API Gateway
cd go-backend/api-gateway && go test ./... -v

# Auth Service
cd go-backend/auth-service && go test ./... -v

# Product Service
cd go-backend/product-service && go test ./... -v

# Order Service
cd go-backend/order-service && go test ./... -v

# Address Service
cd go-backend/address-service && go test ./... -v
```

---

## Gateway — İzole MongoDB Veritabanı

Gateway, kendi MongoDB veritabanını kullanır (`eticaret_gateway`). Mikroservislerin veritabanlarından tamamen izoledir.

```mermaid
graph LR
    GW["API Gateway"] -->|"LogRequest()"| GWDB[("eticaret_gateway\nrequest_logs")]
    AUTH_SVC["Auth Service"] --> AUTHDB[("eticaret_auth\nusers")]
    PRODUCT_SVC["Product Service"] --> PRODDB[("eticaret_products\nproducts")]
    ORDER_SVC["Order Service"] --> ORDERDB[("eticaret_orders\norders")]
    ADDRESS_SVC["Address Service"] --> ADDRDB[("eticaret_addresses\naddresses")]
```

**Son logları görüntüle:**
```bash
curl http://localhost:8080/gateway/logs?limit=50
```

---

## k6 Yük Testi

```mermaid
graph TD
    subgraph Smoke["Smoke Test (k6_smoke_test.js)"]
        S1["1 VU — 30 saniye"]
        S2["Health + GET /products"]
        S1 --- S2
    end

    subgraph Load["Load Test (k6_load_test.js)"]
        L1["50 VU → 100 → 200 → 500 VU"]
        L2["Tüm servisler test edilir"]
        L3["Özel metrikler: auth_duration, product_duration"]
        L1 --- L2 --- L3
    end

    subgraph Stress["Stress Test (k6_stress_test.js)"]
        ST1["500 VU → 1000 VU — kırılma noktası"]
        ST2["Hata oranı takibi"]
        ST1 --- ST2
    end
```

**Çalıştırma:**
```bash
# Docker ile (önerilen)
docker compose --profile loadtest run k6 run /scripts/k6_smoke_test.js
docker compose --profile loadtest run k6 run /scripts/k6_load_test.js
docker compose --profile loadtest run k6 run /scripts/k6_stress_test.js

# Yerel k6 ile
k6 run load-tests/k6_smoke_test.js
```

---

## Proje Klasör Yapısı

```
go-backend/
├── api-gateway/                  # Port 8080 — Tek genel giriş noktası
│   ├── cmd/main.go               # Giriş noktası, route tanımları
│   ├── internal/
│   │   ├── handler/              # Auth/Product/Order/Health handler'ları
│   │   ├── middleware/           # CORS, JWT, RequireRole, RequestLoggerWithStore
│   │   ├── ratelimit/            # Token bucket rate limiter
│   │   └── store/                # Gateway izole MongoDB store
│   ├── Dockerfile
│   └── go.mod
│
├── auth-service/                 # Port 8081
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── handler/              # Register, Login, Profile
│   │   ├── middleware/           # NetworkIsolation, GetUserID/Role/Email
│   │   ├── model/                # User, LoginRequest, AuthResponse
│   │   ├── repository/           # MongoDB erişimi
│   │   └── service/              # bcrypt, JWT üretimi
│   └── tests/                    # Entegrasyon testleri
│
├── product-service/              # Port 8082
│   ├── internal/
│   │   ├── handler/              # CRUD + arama + featured
│   │   ├── middleware/           # NetworkIsolation, RequireAdmin
│   │   ├── model/                # Product, Category, Filter
│   │   ├── repository/           # Filtreli MongoDB sorguları
│   │   └── service/
│   └── tests/
│
├── address-service/              # Port 8083
│   ├── internal/                 # CRUD + sahiplik kontrolü
│   └── tests/
│
├── order-service/                # Port 8084
│   ├── internal/                 # CRUD + stok kontrolü (→ product-service)
│   └── tests/
│
├── cdn-service/                  # Port 8085 — Statik dosya sunucu
│   └── static/                   # Ürün görselleri
│
├── shared/                       # Tüm servisler tarafından kullanılan
│   ├── jwt/                      # Token üretimi ve doğrulaması
│   ├── response/                 # Standart JSON yanıtlar
│   └── logger/                   # Yapılandırılmış JSON logger
│
├── load-tests/                   # k6 yük test scriptleri
│   ├── k6_smoke_test.js
│   ├── k6_load_test.js
│   ├── k6_stress_test.js
│   └── results/
│
└── docker-compose.yml
```

---

## Çalıştırma

### Docker ile (Önerilen)
```bash
cd go-backend
cp .env.example .env      # Değişkenleri düzenle (isteğe bağlı)
docker compose up --build
```

### Yerel Geliştirme
```bash
cd go-backend/shared && go mod download

# Her servis ayrı terminalde:
cd auth-service    && JWT_SECRET=secret go run ./cmd
cd product-service && JWT_SECRET=secret go run ./cmd
cd address-service && JWT_SECRET=secret go run ./cmd
cd order-service   && JWT_SECRET=secret PRODUCT_SERVICE_URL=http://localhost:8082 go run ./cmd
cd api-gateway     && go run ./cmd
```

### Sağlık Kontrolü
```bash
curl http://localhost:8080/health
```

---

## API Endpoint'leri

### Auth (`/auth/*`)
| Method | Path | Auth | Açıklama |
|--------|------|------|----------|
| POST | /auth/register | ❌ | Kayıt |
| POST | /auth/login | ❌ | Giriş → JWT |
| GET | /auth/profile | ✅ JWT | Profil bilgisi |
| PUT | /auth/profile | ✅ JWT | Profil güncelle |

### Ürünler (`/products/*`)
| Method | Path | Auth | Açıklama |
|--------|------|------|----------|
| GET | /products | ❌ | Liste (filtreleme destekli) |
| GET | /products/{id} | ❌ | Ürün detayı |
| GET | /products/slug/{slug} | ❌ | Slug ile detay |
| GET | /products/featured | ❌ | Öne çıkan ürünler |
| GET | /products/search?q= | ❌ | Arama |
| POST | /products | ✅ Admin | Yeni ürün |
| PUT | /products/{id} | ✅ Admin | Ürün güncelle |
| DELETE | /products/{id} | ✅ Admin | Ürün sil |

### Adresler (`/addresses/*`)
| Method | Path | Auth | Açıklama |
|--------|------|------|----------|
| GET | /addresses | ✅ JWT | Adreslerim |
| POST | /addresses | ✅ JWT | Yeni adres |
| GET | /addresses/{id} | ✅ JWT | Adres detayı |
| PUT | /addresses/{id} | ✅ JWT | Adres güncelle |
| DELETE | /addresses/{id} | ✅ JWT | Adres sil |

### Siparişler (`/orders/*`)
| Method | Path | Auth | Açıklama |
|--------|------|------|----------|
| GET | /orders | ✅ JWT | Siparişlerim |
| POST | /orders | ✅ JWT | Sipariş oluştur |
| GET | /orders/{id} | ✅ JWT | Sipariş detayı |
| GET | /orders/number/{no} | ✅ JWT | Sipariş no ile detay |
| POST | /orders/{no}/cancel | ✅ JWT | İptal et |
| PUT | /orders/{id}/status | ✅ Admin | Durum güncelle |

### Gateway
| Method | Path | Auth | Açıklama |
|--------|------|------|----------|
| GET | /health | ❌ | Tüm servislerin durumu |
| GET | /gateway/logs | ❌ | Son N istek logu |

---

## Güvenlik Özellikleri

| Özellik | Açıklama |
|---------|----------|
| **Ağ İzolasyonu** | Mikroservisler dış ağa kapalı, sadece `X-Internal-Secret` ile erişilir |
| **Merkezi JWT** | JWT yalnızca gateway'de doğrulanır, servisler `X-User-*` header'larına güvenir |
| **Rate Limiting** | Token bucket algoritması, dakikada 60 istek (yapılandırılabilir) |
| **CORS** | Gateway seviyesinde tüm origin'ler için kontrol |
| **Rol Kontrolü** | `customer` / `admin` rolleri, her endpoint için ayrı kontrol |
| **Sahiplik Kontrolü** | Adres/sipariş işlemlerinde `X-User-ID` doğrulaması |
| **bcrypt** | Şifreler `cost=12` ile hash'lenir |

---

## PHP'den Go Dönüşüm Özeti

| PHP Bileşeni | Go Karşılığı |
|---|---|
| `AuthController` | `auth-service` |
| `ProductController` | `product-service` |
| `OrderController` | `order-service` |
| `UserController` (adresler) | `address-service` |
| `Session::isLoggedIn()` | JWT middleware (gateway) |
| MySQL | MongoDB |
| PHP Router | API Gateway (reverse proxy) |
| `password_hash()` | `bcrypt.GenerateFromPassword()` |
| Tek sunucu | Docker Compose (6 konteyner) |

### Düzeltilen Güvenlik Açıkları
| Açık | PHP | Go |
|------|-----|-----|
| SQL Injection | `findByEmailVulnerable()` | Tip güvenli MongoDB sorguları |
| IDOR | Adres sahipliği kontrolü yok | Her istekte `X-User-ID` doğrulama |
| SSRF | `file_get_contents($avatarUrl)` | Endpoint kaldırıldı |
| XSS | Arama sorgusu kaçışsız HTML | JSON API, HTML render etmiyor |
