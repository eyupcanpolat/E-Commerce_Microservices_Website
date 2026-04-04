/**
 * k6 Stress Test — Sistemin kırılma noktasını bul
 * Amaç: Rate limiting, hata yönetimi, recovery testi
 * Kullanım: k6 run k6_stress_test.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';

const errorRate       = new Rate('error_rate');
const rateLimitHits   = new Counter('rate_limit_429');
const serviceErrors   = new Counter('service_5xx');

export const options = {
  stages: [
    { duration: '20s', target: 100  },
    { duration: '40s', target: 500  },
    { duration: '40s', target: 1000 }, // Stres seviyesi
    { duration: '20s', target: 0    }, // Recovery
  ],
  thresholds: {
    // Stres testinde eşikler daha gevşek
    http_req_duration: ['p(99)<5000'],
    error_rate: ['rate<0.30'], // %30 tolerans (aşırı yükte beklenen)
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  // Sadece public endpoint — auth gerektirmeyen
  const endpoints = [
    `${BASE_URL}/health`,
    `${BASE_URL}/products`,
    `${BASE_URL}/products/featured`,
    `${BASE_URL}/products/search?q=test`,
  ];

  const url = endpoints[Math.floor(Math.random() * endpoints.length)];
  const res = http.get(url);

  if (res.status === 429) rateLimitHits.add(1);
  if (res.status >= 500) serviceErrors.add(1);

  const ok = check(res, {
    'not server error': (r) => r.status < 500,
  });
  errorRate.add(!ok);

  sleep(0.1);
}

export function handleSummary(data) {
  return {
    'results/stress_summary.json': JSON.stringify(data, null, 2),
  };
}
