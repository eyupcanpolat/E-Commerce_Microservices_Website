/**
 * home.js — Ürün Listeleme ve Sepet Sayfaları
 * Ürünler: listeleme, filtreleme, sayfalama, detay
 * Sepet: localStorage üzerinden yönetim, sipariş oluşturma
 */

// ── SAYFA: Ürün Listesi ─────────────────────────────────────────────────────
function renderHomePage() {
  const root = document.getElementById('app-root');
  root.innerHTML = `
    <div class="container">
      <!-- Hero -->
      <section class="hero">
        <h1><i class="fas fa-bolt" style="color:var(--accent)"></i> Harika Ürünler Keşfet</h1>
        <p>Go mikroservis mimarisi ile güçlendirilmiş güvenli alışveriş platformu.</p>
        <div class="hero-actions">
          <button class="btn btn-primary" id="hero-shop"><i class="fas fa-shopping-bag"></i> Alışverişe Başla</button>
          ${!Auth.isLoggedIn() ? '<button class="btn btn-secondary" id="hero-login"><i class="fas fa-sign-in-alt"></i> Giriş Yap</button>' : ''}
        </div>
      </section>

      <!-- Filtreler -->
      <div class="filter-bar" id="filter-bar">
        <input type="text" id="filter-q" placeholder="🔍 Ürün ara..." style="flex:1;min-width:200px">
        <select id="filter-sort">
          <option value="">Sıralama</option>
          <option value="price_asc">Fiyat: Düşükten Yükseğe</option>
          <option value="price_desc">Fiyat: Yüksekten Düşüğe</option>
          <option value="newest">En Yeni</option>
          <option value="popular">En Popüler</option>
        </select>
        <select id="filter-stock">
          <option value="">Tüm Stok</option>
          <option value="1">Stokta Var</option>
        </select>
        <button class="btn btn-primary btn-sm" id="filter-apply"><i class="fas fa-filter"></i> Filtrele</button>
        <button class="btn btn-secondary btn-sm" id="filter-reset">Sıfırla</button>
      </div>

      <!-- Ürün grid -->
      <div class="section-header">
        <h2 class="section-title" id="products-section-title">Tüm Ürünler</h2>
        <span id="products-count" style="color:var(--text-muted);font-size:13px"></span>
      </div>
      <div id="products-grid" class="products-grid">
        <div class="page-loading"><div class="spinner"></div></div>
      </div>
      <div id="products-pagination"></div>
    </div>
  `;

  // Event binders
  document.getElementById('hero-shop')?.addEventListener('click', () => {
    document.getElementById('filter-bar')?.scrollIntoView({ behavior: 'smooth' });
  });
  document.getElementById('hero-login')?.addEventListener('click', () => App.navigate('login'));
  document.getElementById('filter-apply').addEventListener('click', () => loadProducts(1));
  document.getElementById('filter-reset').addEventListener('click', () => {
    document.getElementById('filter-q').value = '';
    document.getElementById('filter-sort').value = '';
    document.getElementById('filter-stock').value = '';
    loadProducts(1);
  });
  document.getElementById('filter-q').addEventListener('keydown', e => {
    if (e.key === 'Enter') loadProducts(1);
  });

  loadProducts(1);
}

async function loadProducts(page = 1) {
  const grid = document.getElementById('products-grid');
  grid.innerHTML = '<div class="page-loading"><div class="spinner"></div></div>';

  const params = {
    page,
    q:        document.getElementById('filter-q')?.value.trim() || '',
    sort:     document.getElementById('filter-sort')?.value || '',
    in_stock: document.getElementById('filter-stock')?.value === '1' ? true : false
  };

  try {
    const result = await API.products.list(params);
    const { data = [], total = 0, page: pg = 1, total_pages = 1, per_page = 12 } = result;

    document.getElementById('products-section-title').textContent =
      params.q ? `"${params.q}" için sonuçlar` : 'Tüm Ürünler';
    document.getElementById('products-count').textContent =
      total > 0 ? `${total} ürün bulundu` : '';

    if (data.length === 0) {
      grid.innerHTML = `
        <div class="empty-state" style="grid-column:1/-1">
          <i class="fas fa-box-open"></i>
          <h2>Ürün bulunamadı</h2>
          <p>Farklı arama kriterleri deneyin.</p>
          <button class="btn btn-secondary" onclick="document.getElementById('filter-reset').click()">
            Filtreleri Temizle
          </button>
        </div>`;
      document.getElementById('products-pagination').innerHTML = '';
      return;
    }

    grid.innerHTML = data.map(p => productCardHTML(p)).join('');
    renderPagination('products-pagination', pg, total_pages, loadProducts);

    // Bind cart buttons
    grid.querySelectorAll('.add-to-cart').forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const id = parseInt(btn.dataset.id);
        const p = data.find(x => x.id === id);
        if (!p) return;
        Cart.add(p);
        App.updateCartBadge();
        showToast(`"${p.name}" sepete eklendi`, 'success');
      });
    });

    // Bind card click → detail
    grid.querySelectorAll('.product-card').forEach(card => {
      card.addEventListener('click', () => {
        const id = card.dataset.id;
        App.navigate('product', { id });
      });
    });

  } catch (err) {
    grid.innerHTML = `
      <div class="empty-state" style="grid-column:1/-1">
        <i class="fas fa-exclamation-triangle" style="color:var(--danger)"></i>
        <h2>Ürünler yüklenemedi</h2>
        <p>${err.message || 'Bağlantı hatası. API Gateway çalışıyor mu?'}</p>
        <button class="btn btn-primary" onclick="loadProducts(1)">Tekrar Dene</button>
      </div>`;
  }
}

function productCardHTML(p) {
  const price = p.sale_price ? p.sale_price : p.price;
  const isOut = p.stock_status === 'out_of_stock';
  return `
    <div class="product-card" data-id="${p.id}">
      ${p.image_url
        ? `<img class="product-img" src="${p.image_url}" alt="${p.name}" loading="lazy" onerror="this.parentElement.innerHTML = this.parentElement.innerHTML.replace(this.outerHTML,'<div class=\\'product-img-placeholder\\'><i class=\\'fas fa-image\\'></i></div>')">`
        : `<div class="product-img-placeholder"><i class="fas fa-image"></i></div>`}
      <div class="product-body">
        <div class="product-name">${p.name}</div>
        <div class="product-price">
          <span class="price-current">${formatPrice(price)}</span>
          ${p.sale_price ? `<span class="price-original">${formatPrice(p.price)}</span>` : ''}
        </div>
        <div class="product-stock">${stockBadge(p.stock_status)}</div>
        <button class="add-to-cart" data-id="${p.id}" ${isOut ? 'disabled' : ''}>
          <i class="fas fa-cart-plus"></i>
          ${isOut ? 'Stokta Yok' : 'Sepete Ekle'}
        </button>
      </div>
    </div>`;
}

function renderPagination(containerId, current, total, onPage) {
  const el = document.getElementById(containerId);
  if (total <= 1) { el.innerHTML = ''; return; }

  let html = `<div class="pagination">`;
  html += `<button class="page-btn" ${current === 1 ? 'disabled' : ''} onclick="${onPage.name}(${current - 1})">
    <i class="fas fa-chevron-left"></i></button>`;

  for (let i = 1; i <= total; i++) {
    if (total > 7 && i > 2 && i < total - 1 && Math.abs(i - current) > 1) {
      if (i === 3 || i === total - 2) html += `<span class="page-btn" style="cursor:default">…</span>`;
      continue;
    }
    html += `<button class="page-btn ${i === current ? 'active' : ''}" onclick="${onPage.name}(${i})">${i}</button>`;
  }

  html += `<button class="page-btn" ${current === total ? 'disabled' : ''} onclick="${onPage.name}(${current + 1})">
    <i class="fas fa-chevron-right"></i></button>`;
  html += '</div>';
  el.innerHTML = html;
}

// ── SAYFA: Ürün Detay ──────────────────────────────────────────────────────
async function renderProductPage(params = {}) {
  const root = document.getElementById('app-root');
  root.innerHTML = `<div class="container"><div class="page-loading"><div class="spinner"></div></div></div>`;

  try {
    const product = await API.products.get(params.id);
    const price = product.sale_price || product.price;

    root.innerHTML = `
      <div class="container">
        <div style="margin-bottom:20px">
          <button class="btn btn-secondary btn-sm" id="back-btn">
            <i class="fas fa-arrow-left"></i> Ürünlere Dön
          </button>
        </div>
        <div class="product-detail">
          <div>
            ${product.image_url
              ? `<img class="detail-img" src="${product.image_url}" alt="${product.name}">`
              : `<div class="detail-img" style="display:flex;align-items:center;justify-content:center;font-size:80px;color:var(--text-dim)"><i class="fas fa-image"></i></div>`}
          </div>
          <div>
            <h1 class="detail-name">${product.name}</h1>
            <div class="detail-price">
              <span class="price-current">${formatPrice(price)}</span>
              ${product.sale_price ? `<span class="price-original">${formatPrice(product.price)}</span>` : ''}
            </div>
            <div style="margin-bottom:16px">${stockBadge(product.stock_status)}</div>
            ${product.description ? `<p class="detail-desc">${product.description}</p>` : ''}
            ${product.sku ? `<p style="font-size:12px;color:var(--text-muted);margin-bottom:16px">SKU: ${product.sku}</p>` : ''}
            <button class="btn btn-primary detail-add" id="detail-add-btn"
              ${product.stock_status === 'out_of_stock' ? 'disabled' : ''}>
              <i class="fas fa-cart-plus"></i>
              ${product.stock_status === 'out_of_stock' ? 'Stokta Yok' : 'Sepete Ekle'}
            </button>
            <button class="btn btn-secondary" style="width:100%;margin-top:10px;justify-content:center" id="go-cart-btn">
              <i class="fas fa-shopping-cart"></i> Sepete Git
            </button>
          </div>
        </div>
      </div>`;

    document.getElementById('back-btn').addEventListener('click', () => App.navigate('home'));
    document.getElementById('detail-add-btn')?.addEventListener('click', () => {
      Cart.add(product);
      App.updateCartBadge();
      showToast(`"${product.name}" sepete eklendi`, 'success');
    });
    document.getElementById('go-cart-btn')?.addEventListener('click', () => App.navigate('cart'));

  } catch (err) {
    root.innerHTML = `
      <div class="container">
        <div class="empty-state">
          <i class="fas fa-exclamation-circle" style="color:var(--danger)"></i>
          <h2>Ürün bulunamadı</h2>
          <p>${err.message}</p>
          <button class="btn btn-primary" onclick="App.navigate('home')">Ana Sayfaya Dön</button>
        </div>
      </div>`;
  }
}

// ── SAYFA: Sepet ───────────────────────────────────────────────────────────
function renderCartPage() {
  const root = document.getElementById('app-root');
  const items = Cart.get();

  if (items.length === 0) {
    root.innerHTML = `
      <div class="container">
        <div class="cart-empty">
          <i class="fas fa-shopping-cart"></i>
          <h2 style="margin-bottom:8px">Sepetiniz boş</h2>
          <p>Ürünleri inceleyerek alışverişe başlayın.</p>
          <button class="btn btn-primary" style="margin-top:20px" onclick="App.navigate('home')">
            <i class="fas fa-arrow-left"></i> Ürünlere Git
          </button>
        </div>
      </div>`;
    return;
  }

  const subtotal = Cart.total();
  const shipping = subtotal < 500 ? 29.90 : 0;
  const tax = subtotal * 0.18;
  const total = subtotal + shipping + tax;

  root.innerHTML = `
    <div class="container">
      <div class="section-header">
        <h2 class="section-title">Sepetim <span style="font-size:14px;font-weight:500;color:var(--text-muted)">(${Cart.count()} ürün)</span></h2>
        <button class="btn btn-danger btn-sm" id="clear-cart"><i class="fas fa-trash"></i> Sepeti Boşalt</button>
      </div>
      <div class="cart-layout">
        <div class="cart-items" id="cart-items">
          ${items.map(item => cartItemHTML(item)).join('')}
        </div>
        <div class="cart-summary">
          <h3><i class="fas fa-receipt" style="color:var(--accent)"></i> Sipariş Özeti</h3>
          <div class="summary-row"><span>Ara Toplam</span><span>${formatPrice(subtotal)}</span></div>
          <div class="summary-row">
            <span>Kargo ${shipping === 0 ? '<span style="color:var(--success);font-size:11px"> (Ücretsiz)</span>' : ''}</span>
            <span>${shipping === 0 ? 'ÜCRETSİZ' : formatPrice(shipping)}</span>
          </div>
          <div class="summary-row"><span>KDV (%18)</span><span>${formatPrice(tax)}</span></div>
          <div class="summary-total"><span>Toplam</span><span>${formatPrice(total)}</span></div>
          ${subtotal < 500 ? `<p style="font-size:12px;color:var(--warning);margin-top:8px">
            <i class="fas fa-info-circle"></i> ${formatPrice(500 - subtotal)} daha ekleyin, kargo ücretsiz!
          </p>` : ''}
          <button class="btn btn-primary cart-checkout" id="checkout-btn">
            <i class="fas fa-lock"></i> Siparişi Tamamla
          </button>
          <button class="btn btn-secondary" style="width:100%;justify-content:center;margin-top:8px" onclick="App.navigate('home')">
            <i class="fas fa-arrow-left"></i> Alışverişe Devam Et
          </button>
        </div>
      </div>
    </div>`;

  bindCartEvents();

  document.getElementById('checkout-btn').addEventListener('click', () => {
    if (!Auth.isLoggedIn()) {
      showToast('Sipariş vermek için giriş yapmalısınız', 'warning');
      App.navigate('login');
      return;
    }
    renderCheckoutSection();
  });

  document.getElementById('clear-cart').addEventListener('click', () => {
    if (confirm('Sepetinizi boşaltmak istediğinizden emin misiniz?')) {
      Cart.clear();
      App.updateCartBadge();
      renderCartPage();
    }
  });
}

function cartItemHTML(item) {
  return `
    <div class="cart-item" data-id="${item.product_id}">
      ${item.image_url
        ? `<img class="cart-item-img" src="${item.image_url}" alt="${item.product_name}">`
        : `<div class="cart-item-img" style="display:flex;align-items:center;justify-content:center;font-size:28px;color:var(--text-dim)"><i class="fas fa-image"></i></div>`}
      <div class="cart-item-info">
        <div class="cart-item-name">${item.product_name}</div>
        <div class="cart-item-price">${formatPrice(item.unit_price * item.quantity)}</div>
        <div style="font-size:12px;color:var(--text-muted)">${formatPrice(item.unit_price)} × ${item.quantity}</div>
      </div>
      <div class="quantity-control">
        <button class="qty-btn qty-minus" data-id="${item.product_id}"><i class="fas fa-minus"></i></button>
        <span class="qty-value">${item.quantity}</span>
        <button class="qty-btn qty-plus" data-id="${item.product_id}"><i class="fas fa-plus"></i></button>
      </div>
      <button class="cart-remove" data-id="${item.product_id}" title="Kaldır">
        <i class="fas fa-trash"></i>
      </button>
    </div>`;
}

function bindCartEvents() {
  document.querySelectorAll('.qty-minus').forEach(btn => {
    btn.addEventListener('click', () => {
      const id = parseInt(btn.dataset.id);
      const item = Cart.get().find(i => i.product_id === id);
      Cart.updateQty(id, (item?.quantity || 1) - 1);
      App.updateCartBadge();
      renderCartPage();
    });
  });
  document.querySelectorAll('.qty-plus').forEach(btn => {
    btn.addEventListener('click', () => {
      const id = parseInt(btn.dataset.id);
      const item = Cart.get().find(i => i.product_id === id);
      Cart.updateQty(id, (item?.quantity || 0) + 1);
      App.updateCartBadge();
      renderCartPage();
    });
  });
  document.querySelectorAll('.cart-remove').forEach(btn => {
    btn.addEventListener('click', () => {
      Cart.remove(parseInt(btn.dataset.id));
      App.updateCartBadge();
      renderCartPage();
    });
  });
}

// ── Checkout ───────────────────────────────────────────────────────────────
async function renderCheckoutSection() {
  let addresses = [];
  try {
    addresses = await API.addresses.list();
  } catch (_) { /* ignore */ }

  const checkoutHTML = `
    <div class="checkout-section" id="checkout-section">
      <h3 class="checkout-title"><i class="fas fa-map-marker-alt" style="color:var(--accent)"></i> Teslimat Bilgileri</h3>
      <div class="form-group">
        <label>Teslimat Adresi</label>
        <select id="checkout-address">
          <option value="">-- Adres seçin --</option>
          ${addresses.map(a => `<option value="${a.id}">${a.title} — ${a.city}</option>`).join('')}
          <option value="new">+ Yeni adres ekle</option>
        </select>
      </div>
      <div class="form-group">
        <label>Kargo Yöntemi</label>
        <select id="checkout-shipping">
          <option value="standard">Standart Kargo (2-5 iş günü)</option>
          <option value="express">Hızlı Kargo (1-2 iş günü)</option>
        </select>
      </div>
      <div class="form-group">
        <label>Ödeme Yöntemi</label>
        <select id="checkout-payment">
          <option value="credit_card">Kredi Kartı</option>
          <option value="debit_card">Banka Kartı</option>
          <option value="bank_transfer">Havale / EFT</option>
        </select>
      </div>
      <div class="form-group" id="checkout-cc-group">
        <label>Kart Numarası (Demo)</label>
        <input type="text" id="checkout-cc" placeholder="1234123412341234" maxlength="16" pattern="\\d{16}">
        <small style="color:var(--text-muted)">16 haneli kart numaranızı girin</small>
      </div>
      <div class="form-group">
        <label>Sipariş Notu (İsteğe Bağlı)</label>
        <textarea id="checkout-note" rows="2" placeholder="Kurye için not..."></textarea>
      </div>
      <button class="btn btn-primary" style="width:100%;justify-content:center;padding:14px" id="place-order-btn">
        <i class="fas fa-check-circle"></i> Siparişi Onayla
      </button>
    </div>`;

  const container = document.querySelector('.cart-layout');
  const existingCheckout = document.getElementById('checkout-section');
  if (existingCheckout) existingCheckout.remove();
  container.insertAdjacentHTML('beforeend', checkoutHTML);
  document.getElementById('checkout-section').scrollIntoView({ behavior: 'smooth' });

  document.getElementById('checkout-address').addEventListener('change', (e) => {
    if (e.target.value === 'new') App.navigate('addresses');
  });

  document.getElementById('checkout-payment').addEventListener('change', (e) => {
    const ccGroup = document.getElementById('checkout-cc-group');
    if (e.target.value === 'bank_transfer') {
      ccGroup.style.display = 'none';
    } else {
      ccGroup.style.display = 'block';
    }
  });

  document.getElementById('place-order-btn').addEventListener('click', placeOrder);
}

async function placeOrder() {
  const btn = document.getElementById('place-order-btn');
  const addressId = parseInt(document.getElementById('checkout-address')?.value || 0);
  if (!addressId) {
    showToast('Lütfen bir teslimat adresi seçin', 'warning');
    return;
  }

  const paymentMethod = document.getElementById('checkout-payment')?.value || 'credit_card';
  if (paymentMethod !== 'bank_transfer') {
    const rawVal = document.getElementById('checkout-cc')?.value || '';
    const ccVal = rawVal.replace(/\s+/g, '');
    if (!ccVal || ccVal.length !== 16 || !/^\d{16}$/.test(ccVal)) {
      showToast('Lütfen geçerli 16 haneli bir kart numarası girin', 'warning');
      return;
    }
  }

  const items = Cart.get().map(i => ({ product_id: i.product_id, quantity: i.quantity }));
  btn.disabled = true;
  btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Sipariş veriliyor...';

  try {
    const order = await API.orders.create({
      shipping_address_id: addressId,
      shipping_method:  document.getElementById('checkout-shipping')?.value || 'standard',
      payment_method:   document.getElementById('checkout-payment')?.value || 'credit_card',
      notes:            document.getElementById('checkout-note')?.value || '',
      items
    });
    Cart.clear();
    App.updateCartBadge();
    showToast(`Siparişiniz alındı! #${order.order_number}`, 'success');
    App.navigate('orders');
  } catch (err) {
    showToast(err.message || 'Sipariş oluşturulamadı', 'error');
    btn.disabled = false;
    btn.innerHTML = '<i class="fas fa-check-circle"></i> Siparişi Onayla';
  }
}
