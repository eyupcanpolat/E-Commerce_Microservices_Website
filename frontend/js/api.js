/**
 * api.js — Merkezi API İstemcisi
 *
 * TÜM istekler buradaki tek `request()` fonksiyonu üzerinden geçer.
 * Endpoint: sadece API Gateway (port 8080). Mikroservisler doğrudan çağrılmaz.
 * JWT: localStorage'dan alınarak her auth gerektiren isteğe eklenir.
 */

const API_BASE = 'http://localhost:8080';

// ── Auth token helpers ──────────────────────────────────────────────────────
const Auth = {
  getToken()      { return localStorage.getItem('token'); },
  setToken(t)     { localStorage.setItem('token', t); },
  removeToken()   { localStorage.removeItem('token'); },

  getUser()       { const u = localStorage.getItem('user'); return u ? JSON.parse(u) : null; },
  setUser(u)      { localStorage.setItem('user', JSON.stringify(u)); },
  removeUser()    { localStorage.removeItem('user'); },

  isLoggedIn()    { return !!this.getToken(); },
  isAdmin()       { const u = this.getUser(); return u && u.role === 'admin'; },

  save(token, user) {
    this.setToken(token);
    this.setUser(user);
  },
  clear() {
    this.removeToken();
    this.removeUser();
  }
};

// ── Cart (localStorage) ─────────────────────────────────────────────────────
const Cart = {
  KEY: 'cart',
  get()       { const c = localStorage.getItem(this.KEY); return c ? JSON.parse(c) : []; },
  save(items) { localStorage.setItem(this.KEY, JSON.stringify(items)); },
  clear()     { localStorage.removeItem(this.KEY); },

  add(product, quantity = 1) {
    const items = this.get();
    const existing = items.find(i => i.product_id === product.id);
    if (existing) {
      existing.quantity += quantity;
    } else {
      items.push({
        product_id:   product.id,
        product_name: product.name,
        product_sku:  product.sku || '',
        unit_price:   product.sale_price || product.price,
        image_url:    product.image_url || '',
        quantity
      });
    }
    this.save(items);
    return items;
  },

  remove(productId) {
    const items = this.get().filter(i => i.product_id !== productId);
    this.save(items);
    return items;
  },

  updateQty(productId, quantity) {
    if (quantity < 1) return this.remove(productId);
    const items = this.get().map(i =>
      i.product_id === productId ? { ...i, quantity } : i
    );
    this.save(items);
    return items;
  },

  count()  { return this.get().reduce((s, i) => s + i.quantity, 0); },
  total()  { return this.get().reduce((s, i) => s + i.unit_price * i.quantity, 0); }
};

// ── Core HTTP helper ────────────────────────────────────────────────────────
/**
 * request(path, options)
 * @param {string}  path     - e.g. '/products', '/auth/login'
 * @param {object}  options  - fetch-style options + { auth: bool }
 * @returns {Promise<any>}   - parsed JSON data or throws on error
 */
async function request(path, { method = 'GET', body, auth = false, headers = {} } = {}) {
  const opts = {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...headers
    }
  };

  // Attach JWT if auth required or token available
  if (auth || Auth.isLoggedIn()) {
    const token = Auth.getToken();
    if (token) opts.headers['Authorization'] = `Bearer ${token}`;
  }

  if (body) opts.body = JSON.stringify(body);

  const res = await fetch(`${API_BASE}${path}`, opts);
  const json = await res.json().catch(() => ({ success: false, error: 'Sunucudan geçersiz yanıt alındı' }));

  if (!res.ok) {
    // Token expired / invalid → auto logout
    if (res.status === 401) {
      Auth.clear();
      if (window.App) window.App.updateNav();
    }
    throw { status: res.status, message: json.error || json.message || 'Bilinmeyen hata' };
  }

  return json.data !== undefined ? json.data : json;
}

// ── API Endpoints ───────────────────────────────────────────────────────────
const API = {

  // AUTH ─────────────────────────────────────────────────────────────────────
  auth: {
    login(email, password) {
      return request('/auth/login', { method: 'POST', body: { email, password } });
    },
    register(data) {
      return request('/auth/register', { method: 'POST', body: data });
    },
    updateProfile(data) {
      return request('/auth/profile', { method: 'PUT', body: data, auth: true });
    }
  },

  // PRODUCTS ─────────────────────────────────────────────────────────────────
  products: {
    list(params = {}) {
      const qs = new URLSearchParams();
      if (params.page)       qs.set('page', params.page);
      if (params.q)          qs.set('q', params.q);
      if (params.sort)       qs.set('sort', params.sort);
      if (params.category)   qs.set('category', params.category);
      if (params.min_price)  qs.set('min_price', params.min_price);
      if (params.max_price)  qs.set('max_price', params.max_price);
      if (params.in_stock)   qs.set('in_stock', '1');
      const q = qs.toString();
      return request(`/products${q ? '?' + q : ''}`);
    },
    get(id)       { return request(`/products/${id}`); },
    featured(n=4) { return request(`/products/featured?limit=${n}`); },
    search(q)     { return request(`/products/search?q=${encodeURIComponent(q)}`); },

    create(data)  { return request('/products', { method: 'POST', body: data, auth: true }); },
    update(id, d) { return request(`/products/${id}`, { method: 'PUT', body: d, auth: true }); },
    delete(id)    { return request(`/products/${id}`, { method: 'DELETE', auth: true }); }
  },

  // ADDRESSES ────────────────────────────────────────────────────────────────
  addresses: {
    list()         { return request('/addresses', { auth: true }); },
    get(id)        { return request(`/addresses/${id}`, { auth: true }); },
    create(data)   { return request('/addresses', { method: 'POST', body: data, auth: true }); },
    update(id, d)  { return request(`/addresses/${id}`, { method: 'PUT', body: d, auth: true }); },
    delete(id)     { return request(`/addresses/${id}`, { method: 'DELETE', auth: true }); }
  },

  // ORDERS ───────────────────────────────────────────────────────────────────
  orders: {
    list()         { return request('/orders', { auth: true }); },
    get(id)        { return request(`/orders/${id}`, { auth: true }); },
    create(data)   { return request('/orders', { method: 'POST', body: data, auth: true }); },
    cancel(num)    { return request(`/orders/${num}/cancel`, { method: 'POST', auth: true }); },
    updateStatus(id, status) {
      return request(`/orders/${id}/status`, { method: 'PUT', body: { status }, auth: true });
    }
  }
};

// ── Utility: format price ──────────────────────────────────────────────────
function formatPrice(n) {
  return new Intl.NumberFormat('tr-TR', { style: 'currency', currency: 'TRY' }).format(n);
}

// ── Utility: format date ───────────────────────────────────────────────────
function formatDate(iso) {
  return new Date(iso).toLocaleDateString('tr-TR', {
    year: 'numeric', month: 'long', day: 'numeric',
    hour: '2-digit', minute: '2-digit'
  });
}

// ── Utility: stock badge HTML ──────────────────────────────────────────────
function stockBadge(status) {
  const map = {
    in_stock:    ['stock-in',  'fa-check-circle',  'Stokta Var'],
    low_stock:   ['stock-low', 'fa-exclamation-circle', 'Son Stok'],
    out_of_stock:['stock-out', 'fa-times-circle',  'Tükendi']
  };
  const [cls, icon, label] = map[status] || map.out_of_stock;
  return `<span class="stock-badge ${cls}"><i class="fas ${icon}"></i>${label}</span>`;
}

// ── Utility: order status badge HTML ──────────────────────────────────────
function statusBadge(status) {
  const map = {
    pending:    ['status-pending',    'fa-clock',           'Alındı'],
    processing: ['status-processing', 'fa-cog fa-spin',     'Hazırlanıyor'],
    shipped:    ['status-shipped',    'fa-truck',           'Yola Çıktı'],
    delivered:  ['status-delivered',  'fa-check-double',    'Teslim Edildi'],
    cancelled:  ['status-cancelled',  'fa-times',           'İptal Edildi'],
    refunded:   ['status-cancelled',  'fa-undo',            'İade Edildi']
  };
  const [cls, icon, label] = map[status] || ['', 'fa-circle', status];
  return `<span class="status-badge ${cls}"><i class="fas ${icon}"></i>${label}</span>`;
}
