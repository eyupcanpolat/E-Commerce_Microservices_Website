/**
 * k6 Load Test — Aşamalı yük testi
 * Amaç: 50, 100, 200, 500 eş zamanlı kullanıcı senaryosu
 * Kullanım: k6 run k6_load_test.js
 * Sonuçları kaydet: k6 run --out json=results/load_results.json k6_load_test.js
 */
import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Özel metrikler
const errorRate    = new Rate('error_rate');
const authDuration = new Trend('auth_duration');
const productDuration = new Trend('product_duration');
const orderDuration   = new Trend('order_duration');
const rateLimitHits   = new Counter('rate_limit_hits');

export const options = {
  // Aşamalı yük: 50 → 100 → 200 → 500 kullanıcı (PDF isterine uygun)
  stages: [
    { duration: '30s', target: 50  }, // Ramping up to 50
    { duration: '60s', target: 50  }, // 50 VU sabit
    { duration: '30s', target: 100 }, // Ramping up to 100
    { duration: '60s', target: 100 }, // 100 VU sabit
    { duration: '30s', target: 200 }, // Ramping up to 200
    { duration: '60s', target: 200 }, // 200 VU sabit
    { duration: '30s', target: 500 }, // Ramping up to 500
    { duration: '60s', target: 500 }, // 500 VU sabit
    { duration: '30s', target: 0   }, // Ramp down
  ],
  thresholds: {
    http_req_duration:      ['p(95)<1000', 'p(99)<2000'],
    http_req_failed:        ['rate<0.05'],   // %5 hata toleransı
    error_rate:             ['rate<0.05'],
    auth_duration:          ['p(95)<800'],
    product_duration:       ['p(95)<500'],
  },
  // Prometheus Remote Write: docker-compose'da K6_PROMETHEUS_RW_SERVER_URL env ile aktifleşir
  // Grafana'da gerçek zamanlı görüntüleme sağlar
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test kullanıcıları — her VU farklı kullanıcı simüle eder
const TEST_USERS = [
  { email: 'test1@example.com', password: 'password123' },
  { email: 'test2@example.com', password: 'password123' },
  { email: 'test3@example.com', password: 'password123' },
];

export function setup() {
  // Test başlamadan önce kullanıcıları kaydet
  for (const user of TEST_USERS) {
    http.post(`${BASE_URL}/auth/register`, JSON.stringify({
      email: user.email,
      password: user.password,
      password_confirm: user.password,
      first_name: 'Load',
      last_name: 'Test',
    }), { headers: { 'Content-Type': 'application/json' } });
  }
  return {};
}

export default function () {
  const user = TEST_USERS[__VU % TEST_USERS.length];
  let token = '';

  // ── 1. AUTH SERVİSİ ─────────────────────────────────────────────────────────
  group('Auth Service', function () {
    const loginStart = Date.now();
    const loginRes = http.post(`${BASE_URL}/auth/login`,
      JSON.stringify({ email: user.email, password: user.password }),
      { headers: { 'Content-Type': 'application/json' } }
    );
    authDuration.add(Date.now() - loginStart);

    if (loginRes.status === 429) { rateLimitHits.add(1); return; }

    const ok = check(loginRes, {
      'login 200': (r) => r.status === 200,
      'token var': (r) => {
        try { return JSON.parse(r.body).data?.token?.length > 0; } catch { return false; }
      },
    });
    errorRate.add(!ok);

    if (loginRes.status === 200) {
      try { token = JSON.parse(loginRes.body).data.token; } catch {}
    }
  });

  sleep(0.5);

  // ── 2. PRODUCT SERVİSİ (public) ─────────────────────────────────────────────
  group('Product Service - Public', function () {
    const start = Date.now();

    // Ürün listesi
    const listRes = http.get(`${BASE_URL}/products?page=1`);
    productDuration.add(Date.now() - start);
    if (listRes.status === 429) { rateLimitHits.add(1); return; }

    check(listRes, { 'products 200': (r) => r.status === 200 });
    errorRate.add(listRes.status !== 200);

    // Öne çıkan ürünler
    const featuredRes = http.get(`${BASE_URL}/products/featured`);
    check(featuredRes, { 'featured 200': (r) => r.status === 200 });

    // Arama
    const searchRes = http.get(`${BASE_URL}/products/search?q=laptop`);
    check(searchRes, { 'search 200': (r) => r.status === 200 });
  });

  sleep(0.5);

  // ── 3. ORDER SERVİSİ (auth gerekli) ─────────────────────────────────────────
  if (token) {
    group('Order Service - Authenticated', function () {
      const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      };

      const start = Date.now();
      const ordersRes = http.get(`${BASE_URL}/orders`, { headers });
      orderDuration.add(Date.now() - start);

      if (ordersRes.status === 429) { rateLimitHits.add(1); return; }

      check(ordersRes, {
        'orders 200': (r) => r.status === 200,
      });
      errorRate.add(ordersRes.status !== 200);
    });

    sleep(0.5);

    // ── 4. ADDRESS SERVİSİ (auth gerekli) ───────────────────────────────────────
    group('Address Service - Authenticated', function () {
      const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      };
      const addrRes = http.get(`${BASE_URL}/addresses`, { headers });
      if (addrRes.status === 429) { rateLimitHits.add(1); return; }
      check(addrRes, { 'addresses 200': (r) => r.status === 200 });
    });
  }

  sleep(1);
}

export function handleSummary(data) {
  const metrics  = data.metrics;
  const duration = metrics.http_req_duration;
  const failed   = metrics.http_req_failed;

  const summary = `
╔══════════════════════════════════════════════════════╗
║           YÜK TESTİ SONUÇLARI (50→500 VU)           ║
╠══════════════════════════════════════════════════════╣
║ Toplam İstek      : ${String(metrics.http_reqs?.values?.count || 0).padStart(10)}                    ║
║ Hata Oranı        : ${String(((failed?.values?.rate || 0) * 100).toFixed(2) + '%').padStart(10)}                    ║
║ Ort. Yanıt Süresi : ${String((duration?.values?.avg || 0).toFixed(0) + 'ms').padStart(10)}                    ║
║ p(90)             : ${String((duration?.values?.['p(90)'] || 0).toFixed(0) + 'ms').padStart(10)}                    ║
║ p(95)             : ${String((duration?.values?.['p(95)'] || 0).toFixed(0) + 'ms').padStart(10)}                    ║
║ p(99)             : ${String((duration?.values?.['p(99)'] || 0).toFixed(0) + 'ms').padStart(10)}                    ║
║ Rate Limit (429)  : ${String(metrics.rate_limit_hits?.values?.count || 0).padStart(10)}                    ║
╚══════════════════════════════════════════════════════╝
Detaylı sonuçlar: results/load_summary.json
Grafana: http://localhost:3000 → k6 Load Testing
`;

  return {
    'results/load_summary.json': JSON.stringify(data, null, 2),
    stdout: summary,
  };
}
