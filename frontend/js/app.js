/**
 * app.js — Uygulama Çekirdeği (SPA Router)
 *
 * Özellikler:
 * - Basit hash-based SPA router
 * - Nav state yönetimi (giriş/çıkış)
 * - Toast notification sistemi
 * - Modal yönetimi
 * - Sepet badge güncellemesi
 */

// ── Globals (api.js öncesi yüklenmeli) ─────────────────────────────────────
window.showToast = function(message, type = 'info', duration = 3500) {
  const container = document.getElementById('toast-container');
  if (!container) return;

  const icons = { success: 'fa-check-circle', error: 'fa-times-circle', warning: 'fa-exclamation-triangle', info: 'fa-info-circle' };
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.innerHTML = `
    <i class="fas ${icons[type] || icons.info} toast-icon"></i>
    <span class="toast-msg">${message}</span>`;

  container.appendChild(toast);
  setTimeout(() => {
    toast.style.transition = 'all 0.3s ease';
    toast.style.opacity = '0';
    toast.style.transform = 'translateX(40px)';
    setTimeout(() => toast.remove(), 300);
  }, duration);
};

window.closeModal = function() {
  document.getElementById('modal-overlay')?.classList.add('hidden');
};

// ── App Router ──────────────────────────────────────────────────────────────
const App = {
  // Route map: name → render function
  routes: {
    home:      ()       => renderHomePage(),
    login:     ()       => renderLoginPage(),
    register:  ()       => renderRegisterPage(),
    orders:    ()       => renderOrdersPage(),
    addresses: ()       => renderAddressesPage(),
    cart:      ()       => renderCartPage(),
    profile:   ()       => renderProfilePage(),
    product:   (p)      => renderProductPage(p),
    'admin-products': () => renderAdminProductsPage()
  },

  navigate(page = 'home', params = {}) {
    window._routeParams = params;
    const fn = this.routes[page];
    if (!fn) { this.routes.home(); return; }

    // Scroll back to top
    window.scrollTo({ top: 0, behavior: 'smooth' });
    closeModal();
    fn(params);
  },

  init() {
    this.updateNav();
    this.updateCartBadge();
    this.bindNavEvents();
    this.bindModalEvents();
    this.bindSearchEvents();

    // Default page
    this.navigate('home');
  },

  // ── Nav ─────────────────────────────────────────────────────────────────
  updateNav() {
    const guestNav = document.getElementById('nav-guest');
    const userNav  = document.getElementById('nav-user');
    const avatarEl = document.getElementById('avatar-initials');
    const dropName = document.getElementById('dropdown-username');

    if (Auth.isLoggedIn()) {
      guestNav?.classList.add('hidden');
      userNav?.classList.remove('hidden');
      const user = Auth.getUser();
      if (user && avatarEl) {
        const initials = ((user.first_name?.[0] || '') + (user.last_name?.[0] || '')).toUpperCase() || '?';
        avatarEl.textContent = initials;
      }
      if (dropName && user) {
        dropName.textContent = `${user.first_name || ''} ${user.last_name || ''}`.trim();
      }
      
      const btnAdmin = document.getElementById('btn-admin-products');
      if (btnAdmin) {
        btnAdmin.style.display = (user && user.role === 'admin') ? 'inline-flex' : 'none';
      }
    } else {
      guestNav?.classList.remove('hidden');
      userNav?.classList.add('hidden');
    }
  },

  updateCartBadge() {
    const badge = document.getElementById('cart-badge');
    if (badge) {
      const count = Cart.count();
      badge.textContent = count;
      badge.style.display = count > 0 ? 'inline' : 'none';
    }
  },

  // ── Nav Event Listeners ─────────────────────────────────────────────────
  bindNavEvents() {
    // Guest nav
    document.getElementById('btn-login')?.addEventListener('click', () => this.navigate('login'));

    // User nav
    document.getElementById('btn-orders')?.addEventListener('click', () => this.navigate('orders'));
    document.getElementById('btn-addresses')?.addEventListener('click', () => this.navigate('addresses'));
    document.getElementById('btn-logout')?.addEventListener('click', () => this.logout());
    document.getElementById('btn-admin-products')?.addEventListener('click', () => this.navigate('admin-products'));

    // Logo / home link
    document.getElementById('nav-home')?.addEventListener('click', (e) => {
      e.preventDefault();
      this.navigate('home');
    });

    // Footer links
    document.getElementById('footer-home')?.addEventListener('click', (e) => { e.preventDefault(); this.navigate('home'); });
    document.getElementById('footer-orders')?.addEventListener('click', (e) => { e.preventDefault(); this.navigate('orders'); });
    document.getElementById('footer-addresses')?.addEventListener('click', (e) => { e.preventDefault(); this.navigate('addresses'); });

    // User dropdown toggle
    document.getElementById('btn-user-menu')?.addEventListener('click', () => {
      document.getElementById('user-dropdown')?.classList.toggle('open');
    });
    document.addEventListener('click', (e) => {
      if (!e.target.closest('.user-menu')) {
        document.getElementById('user-dropdown')?.classList.remove('open');
      }
    });
  },

  bindModalEvents() {
    document.getElementById('modal-close')?.addEventListener('click', closeModal);
    document.getElementById('modal-overlay')?.addEventListener('click', (e) => {
      if (e.target === document.getElementById('modal-overlay')) closeModal();
    });
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') closeModal();
    });
  },

  bindSearchEvents() {
    document.getElementById('search-form')?.addEventListener('submit', (e) => {
      e.preventDefault();
      const q = document.getElementById('search-input')?.value.trim();
      if (!q) { this.navigate('home'); return; }
      this.navigate('home');
      // Wait for home page to render then fill search
      setTimeout(() => {
        const filterQ = document.getElementById('filter-q');
        if (filterQ) { filterQ.value = q; loadProducts(1); }
      }, 100);
    });
  },

  logout() {
    Auth.clear();
    Cart.clear();
    this.updateNav();
    this.updateCartBadge();
    showToast('Çıkış yapıldı. Görüşürüz!', 'info');
    this.navigate('home');
  }
};

// ── Init on DOM Ready ───────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  App.init();
});
