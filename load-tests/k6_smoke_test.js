/**
 * k6 Smoke Test — Temel sağlık kontrolü
 * Amaç: Sistem düzgün çalışıyor mu? Minimum yük ile doğrula.
 * Kullanım: k6 run k6_smoke_test.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('error_rate');

export const options = {
  vus: 1,          // 1 sanal kullanıcı
  duration: '30s', // 30 saniye
  thresholds: {
    http_req_duration: ['p(95)<500'], // %95 istek 500ms altında
    error_rate: ['rate<0.01'],        // Hata oranı %1 altında
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  // Health check
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health status 200': (r) => r.status === 200,
    'health ok': (r) => JSON.parse(r.body).status === 'ok',
  });
  errorRate.add(healthRes.status !== 200);

  // Public ürün listesi
  const productsRes = http.get(`${BASE_URL}/products`);
  check(productsRes, {
    'products status 200': (r) => r.status === 200,
  });
  errorRate.add(productsRes.status !== 200);

  sleep(1);
}
