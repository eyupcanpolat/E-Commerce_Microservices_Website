/**
 * auth.js — Kullanıcı Giriş ve Kayıt Sayfaları
 * JWT localStorage'a kaydedilir. Tüm API çağrıları Gateway üzerinden geçer.
 */

// ── SAYFA: Login ────────────────────────────────────────────────────────────
function renderLoginPage() {
  if (Auth.isLoggedIn()) { App.navigate('home'); return; }

  const root = document.getElementById('app-root');
  root.innerHTML = `
    <div class="container" style="padding-top:40px">
      <div class="auth-container">
        <div class="auth-logo">
          <div class="auth-logo-icon"><i class="fas fa-shopping-bag"></i></div>
          <h1 class="auth-title">Hoşgeldiniz</h1>
          <p class="auth-subtitle">Hesabınıza giriş yapın</p>
        </div>
        <div class="form-error" id="login-error"></div>
        <form class="auth-form" id="login-form" novalidate>
          <div class="form-group">
            <label>E-posta Adresi</label>
            <input type="email" id="login-email" placeholder="ornek@email.com" autocomplete="email" required>
          </div>
          <div class="form-group">
            <label>Şifre</label>
            <input type="password" id="login-password" placeholder="••••••••" autocomplete="current-password" required>
          </div>
          <button type="submit" class="btn btn-primary auth-submit" id="login-btn">
            <i class="fas fa-sign-in-alt"></i> Giriş Yap
          </button>
        </form>
        <div class="auth-switch">
          Hesabınız yok mu?
          <a id="goto-register">Kayıt Ol</a>
        </div>
        <div style="margin-top:20px;padding:14px;background:rgba(108,99,255,0.08);border:1px solid rgba(108,99,255,0.2);border-radius:10px;font-size:12px;color:var(--text-muted)">
          <strong style="color:var(--accent)"><i class="fas fa-info-circle"></i> Demo Hesapları</strong><br>
          Admin: <code>admin@eticaret.com</code> / <code>password123</code><br>
          Müşteri: <code>ahmet@example.com</code> / <code>password123</code>
        </div>
      </div>
    </div>`;

  document.getElementById('goto-register').addEventListener('click', () => App.navigate('register'));

  document.getElementById('login-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const errEl = document.getElementById('login-error');
    const btn = document.getElementById('login-btn');
    errEl.classList.remove('show');

    const email    = document.getElementById('login-email').value.trim();
    const password = document.getElementById('login-password').value;

    if (!email || !password) {
      errEl.textContent = 'E-posta ve şifre zorunludur.';
      errEl.classList.add('show');
      return;
    }

    btn.disabled = true;
    btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Giriş yapılıyor...';

    try {
      const data = await API.auth.login(email, password);
      // data: { token, expires_in, user: { id, email, first_name, last_name, role } }
      Auth.save(data.token, data.user);
      App.updateNav();
      App.updateCartBadge();
      showToast(`Hoşgeldiniz, ${data.user.first_name}!`, 'success');
      App.navigate('home');
    } catch (err) {
      errEl.textContent = err.message || 'Giriş başarısız. Bilgilerinizi kontrol edin.';
      errEl.classList.add('show');
      btn.disabled = false;
      btn.innerHTML = '<i class="fas fa-sign-in-alt"></i> Giriş Yap';
    }
  });
}

// ── SAYFA: Register ─────────────────────────────────────────────────────────
function renderRegisterPage() {
  if (Auth.isLoggedIn()) { App.navigate('home'); return; }

  const root = document.getElementById('app-root');
  root.innerHTML = `
    <div class="container" style="padding-top:40px">
      <div class="auth-container">
        <div class="auth-logo">
          <div class="auth-logo-icon"><i class="fas fa-user-plus"></i></div>
          <h1 class="auth-title">Kayıt Ol</h1>
          <p class="auth-subtitle">Yeni hesap oluşturun</p>
        </div>
        <div class="form-error" id="register-error"></div>
        <form class="auth-form" id="register-form" novalidate>
          <div class="form-grid-2">
            <div class="form-group">
              <label>Ad <span style="color:var(--danger)">*</span></label>
              <input type="text" id="reg-firstname" placeholder="Adınız" autocomplete="given-name" required>
            </div>
            <div class="form-group">
              <label>Soyad <span style="color:var(--danger)">*</span></label>
              <input type="text" id="reg-lastname" placeholder="Soyadınız" autocomplete="family-name" required>
            </div>
          </div>
          <div class="form-group">
            <label>E-posta <span style="color:var(--danger)">*</span></label>
            <input type="email" id="reg-email" placeholder="ornek@email.com" autocomplete="email" required>
          </div>
          <div class="form-group">
            <label>Telefon</label>
            <input type="tel" id="reg-phone" placeholder="05XX XXX XXXX" autocomplete="tel">
          </div>
          <div class="form-group">
            <label>Şifre <span style="color:var(--danger)">*</span></label>
            <input type="password" id="reg-password" placeholder="En az 8 karakter" autocomplete="new-password" required>
          </div>
          <div class="form-group">
            <label>Şifre Tekrar <span style="color:var(--danger)">*</span></label>
            <input type="password" id="reg-password-confirm" placeholder="Şifrenizi tekrar girin" autocomplete="new-password" required>
          </div>
          <button type="submit" class="btn btn-primary auth-submit" id="register-btn">
            <i class="fas fa-user-plus"></i> Kayıt Ol
          </button>
        </form>
        <div class="auth-switch">
          Hesabınız var mı?
          <a id="goto-login">Giriş Yap</a>
        </div>
      </div>
    </div>`;

  document.getElementById('goto-login').addEventListener('click', () => App.navigate('login'));

  document.getElementById('register-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const errEl = document.getElementById('register-error');
    const btn = document.getElementById('register-btn');
    errEl.classList.remove('show');

    const firstName       = document.getElementById('reg-firstname').value.trim();
    const lastName        = document.getElementById('reg-lastname').value.trim();
    const email           = document.getElementById('reg-email').value.trim();
    const phone           = document.getElementById('reg-phone').value.trim();
    const password        = document.getElementById('reg-password').value;
    const passwordConfirm = document.getElementById('reg-password-confirm').value;

    // Client-side validation
    if (!firstName || !lastName || !email || !password) {
      errEl.textContent = 'Lütfen tüm zorunlu alanları doldurun.';
      errEl.classList.add('show');
      return;
    }
    if (password.length < 8) {
      errEl.textContent = 'Şifre en az 8 karakter olmalıdır.';
      errEl.classList.add('show');
      return;
    }
    if (password !== passwordConfirm) {
      errEl.textContent = 'Şifreler eşleşmiyor.';
      errEl.classList.add('show');
      return;
    }

    btn.disabled = true;
    btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Kaydediliyor...';

    try {
      const data = await API.auth.register({
        first_name:       firstName,
        last_name:        lastName,
        email,
        phone,
        password,
        password_confirm: passwordConfirm
      });
      Auth.save(data.token, data.user);
      App.updateNav();
      showToast(`Hoşgeldiniz, ${data.user.first_name}! Hesabınız oluşturuldu.`, 'success');
      App.navigate('home');
    } catch (err) {
      errEl.textContent = err.message || 'Kayıt başarısız.';
      errEl.classList.add('show');
      btn.disabled = false;
      btn.innerHTML = '<i class="fas fa-user-plus"></i> Kayıt Ol';
    }
  });
}
