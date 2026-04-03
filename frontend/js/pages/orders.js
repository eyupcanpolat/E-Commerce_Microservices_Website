/**
 * orders.js — Kullanıcı Sipariş Listeleme Sayfası
 * JWT zorunlu. Kullanıcı sadece kendi siparişlerini görür.
 */

async function renderOrdersPage() {
  if (!Auth.isLoggedIn()) {
    showToast('Siparişlerinizi görmek için giriş yapmalısınız', 'warning');
    App.navigate('login');
    return;
  }

  const root = document.getElementById('app-root');
  root.innerHTML = `
    <div class="container">
      <div class="section-header">
        <h1 class="section-title">Siparişlerim</h1>
        <button class="btn btn-secondary btn-sm" onclick="App.navigate('home')">
          <i class="fas fa-shopping-bag"></i> Alışverişe Devam Et
        </button>
      </div>
      <div id="orders-list">
        <div class="page-loading"><div class="spinner"></div></div>
      </div>
    </div>`;

  try {
    const orders = await API.orders.list();
    const container = document.getElementById('orders-list');

    if (!orders || orders.length === 0) {
      container.innerHTML = `
        <div class="empty-state">
          <i class="fas fa-box-open"></i>
          <h2>Henüz siparişiniz yok</h2>
          <p>İlk siparişinizi verin, burada görüntüleyin.</p>
          <button class="btn btn-primary" onclick="App.navigate('home')">
            <i class="fas fa-shopping-bag"></i> Alışverişe Başla
          </button>
        </div>`;
      return;
    }

    // En yeni sipariş önce
    const sorted = [...orders].sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
    container.innerHTML = `<div class="orders-list">${sorted.map(o => orderCardHTML(o)).join('')}</div>`;

    // Cancel buttons
    container.querySelectorAll('.cancel-order-btn').forEach(btn => {
      btn.addEventListener('click', async () => {
        const num = btn.dataset.num;
        if (!confirm(`Sipariş #${num} iptal edilsin mi?`)) return;
        btn.disabled = true;
        btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i>';
        try {
          await API.orders.cancel(num);
          showToast('Sipariş iptal edildi', 'success');
          renderOrdersPage();
        } catch (err) {
          showToast(err.message, 'error');
          btn.disabled = false;
          btn.innerHTML = '<i class="fas fa-times"></i> İptal Et';
        }
      });
    });

    // Admin: status update
    if (Auth.isAdmin()) {
      container.querySelectorAll('.status-select').forEach(sel => {
        sel.addEventListener('change', async () => {
          const id = parseInt(sel.dataset.id);
          try {
            await API.orders.updateStatus(id, sel.value);
            showToast('Sipariş durumu güncellendi', 'success');
            renderOrdersPage();
          } catch (err) {
            showToast(err.message, 'error');
          }
        });
      });
    }

  } catch (err) {
    document.getElementById('orders-list').innerHTML = `
      <div class="empty-state">
        <i class="fas fa-exclamation-triangle" style="color:var(--danger)"></i>
        <h2>Siparişler yüklenemedi</h2>
        <p>${err.message}</p>
        <button class="btn btn-primary" onclick="renderOrdersPage()">Tekrar Dene</button>
      </div>`;
  }
}

function orderCardHTML(order) {
  const canCancel = ['pending', 'processing'].includes(order.status);

  return `
    <div class="order-card">
      <div class="order-header">
        <div>
          <div class="order-num"># ${order.order_number}</div>
          <div class="order-date"><i class="fas fa-calendar-alt" style="color:var(--text-muted)"></i> ${formatDate(order.created_at)}</div>
        </div>
        <div style="display:flex;align-items:center;gap:10px">
          ${statusBadge(order.status)}
          ${Auth.isAdmin() ? `
            <select class="status-select" data-id="${order.id}"
              style="background:rgba(255,255,255,0.05);border:1px solid var(--border);
                     color:var(--text);padding:4px 8px;border-radius:6px;font-size:12px;
                     font-family:inherit;cursor:pointer;">
              <option value="pending"    ${order.status === 'pending'    ? 'selected' : ''}>Alındı</option>
              <option value="processing" ${order.status === 'processing' ? 'selected' : ''}>Hazırlanıyor</option>
              <option value="shipped"    ${order.status === 'shipped'    ? 'selected' : ''}>Yola Çıktı</option>
              <option value="delivered"  ${order.status === 'delivered'  ? 'selected' : ''}>Teslim Edildi</option>
              <option value="cancelled"  ${order.status === 'cancelled'  ? 'selected' : ''}>İptal Edildi</option>
            </select>` : ''}
        </div>
      </div>
      <div class="order-body">
        <div class="order-items-list">
          ${(order.items || []).map(item => `
            <div class="order-item-row">
              <span>${item.product_name} <span style="color:var(--text-muted)">× ${item.quantity}</span></span>
              <span>${formatPrice(item.total_price)}</span>
            </div>`).join('')}
        </div>
        <div class="order-total">
          <span>Toplam</span>
          <span>${formatPrice(order.total)}</span>
        </div>
        ${order.shipping_method ? `
          <div style="margin-top:10px;font-size:12px;color:var(--text-muted)">
            <i class="fas fa-truck"></i> ${order.shipping_method === 'express' ? 'Hızlı Kargo' : 'Standart Kargo'}
            &nbsp;&nbsp;
            <i class="fas fa-credit-card"></i> ${paymentLabel(order.payment_method)}
          </div>` : ''}
        <div class="order-actions">
          ${canCancel ? `
            <button class="btn btn-danger btn-sm cancel-order-btn" data-num="${order.order_number}">
              <i class="fas fa-times"></i> İptal Et
            </button>` : ''}
          <button class="btn btn-secondary btn-sm" onclick="App.navigate('home')">
            <i class="fas fa-redo"></i> Tekrar Sipariş
          </button>
        </div>
      </div>
    </div>`;
}

function paymentLabel(method) {
  const labels = {
    credit_card:   'Kredi Kartı',
    debit_card:    'Banka Kartı',
    bank_transfer: 'Havale/EFT'
  };
  return labels[method] || method || 'Belirtilmemiş';
}
