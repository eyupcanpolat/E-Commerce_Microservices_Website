/**
 * profile.js — Kullanıcı Profil Sayfası
 *
 * Profil bilgileri güncelleme (isim, soyisim, parola)
 * Adres Yönetimi ve Sipariş Geçmişi
 */

async function renderProfilePage() {
  if (!Auth.isLoggedIn()) {
    showToast('Profilinizi görüntülemek için giriş yapmalısınız', 'warning');
    App.navigate('login');
    return;
  }

  const root = document.getElementById('app-root');
  root.innerHTML = `
    <div class="container">
      <div class="section-header">
        <h2 class="section-title"><i class="fas fa-user-circle" style="color:var(--accent)"></i> Profilim</h2>
      </div>

      <div class="profile-layout" style="display:grid;grid-template-columns:1fr 2fr;gap:2rem;">
        
        <!-- Sol Menü -->
        <div class="profile-sidebar" style="background:#151821;padding:20px;border-radius:12px;border:1px solid var(--border);height:fit-content;">
          <ul style="list-style:none;padding:0;margin:0;">
            <li style="margin-bottom:10px;">
              <button class="btn btn-secondary" style="width:100%;justify-content:flex-start" id="tab-info">
                <i class="fas fa-id-card"></i> Profil Bilgilerim
              </button>
            </li>
            <li style="margin-bottom:10px;">
              <button class="btn btn-secondary" style="width:100%;justify-content:flex-start" id="tab-addresses">
                <i class="fas fa-map-marker-alt"></i> Adreslerim
              </button>
            </li>
            <li>
              <button class="btn btn-secondary" style="width:100%;justify-content:flex-start" id="tab-orders">
                <i class="fas fa-box"></i> Siparişlerim
              </button>
            </li>
          </ul>
        </div>

        <!-- Sağ İçerik -->
        <div class="profile-content" id="profile-content">
          <!-- Dinamik olarak doldurulacak -->
        </div>

      </div>
    </div>
  `;

  document.getElementById('tab-info').addEventListener('click', () => loadProfileInfo());
  document.getElementById('tab-addresses').addEventListener('click', () => loadProfileAddresses());
  document.getElementById('tab-orders').addEventListener('click', () => loadProfileOrders());

  // Default tab
  loadProfileInfo();
}

/** ── TABS ── */

function loadProfileInfo() {
  const content = document.getElementById('profile-content');
  const user = Auth.getUser();

  content.innerHTML = `
    <div class="card" style="padding:20px;background:#151821;border-radius:12px;border:1px solid var(--border);">
      <h3 style="margin-bottom:15px;color:var(--text);font-size:18px;">Profil Bilgilerini Güncelle</h3>
      <form id="profile-update-form">
        <div class="form-row" style="display:grid;grid-template-columns:1fr 1fr;gap:15px;">
          <div class="form-group">
            <label>Ad</label>
            <input type="text" id="prof-firstname" value="${user.first_name || ''}" required>
          </div>
          <div class="form-group">
            <label>Soyad</label>
            <input type="text" id="prof-lastname" value="${user.last_name || ''}" required>
          </div>
        </div>
        <div class="form-group">
          <label>E-posta (Değiştirilemez)</label>
          <input type="email" value="${user.email || ''}" disabled style="opacity:0.5">
        </div>
        <div class="form-group">
          <label>Yeni Parola (Değiştirmek istemiyorsanız boş bırakın)</label>
          <input type="password" id="prof-password" placeholder="••••••••">
        </div>
        <button type="submit" class="btn btn-primary"><i class="fas fa-save"></i> Güncelle</button>
      </form>
    </div>
  `;

  document.getElementById('profile-update-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const btn = e.target.querySelector('button');
    btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Güncelleniyor...';
    btn.disabled = true;

    const data = {
      first_name: document.getElementById('prof-firstname').value.trim(),
      last_name: document.getElementById('prof-lastname').value.trim()
    };
    const pwd = document.getElementById('prof-password').value;
    if (pwd) data.password = pwd;

    try {
      const updatedUser = await API.auth.updateProfile(data);
      // Update local storage user info
      const curr = Auth.getUser();
      Auth.setUser({ ...curr, first_name: updatedUser.first_name, last_name: updatedUser.last_name });
      App.updateNav();
      showToast('Profiliniz başarıyla güncellendi', 'success');
      document.getElementById('prof-password').value = '';
    } catch (err) {
      showToast(err.message || 'Güncelleme başarısız', 'error');
    } finally {
      btn.innerHTML = '<i class="fas fa-save"></i> Güncelle';
      btn.disabled = false;
    }
  });
}

async function loadProfileAddresses() {
  const content = document.getElementById('profile-content');
  content.innerHTML = '<div class="spinner"></div>';

  try {
    const addresses = await API.addresses.list();
    let addrHtml = `
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:15px;">
        <h3 style="color:var(--text);font-size:18px;">Kayıtlı Adreslerim</h3>
        <button class="btn btn-primary btn-sm" id="prof-add-addr"><i class="fas fa-plus"></i> Yeni Ekle</button>
      </div>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:15px;">
    `;

    if(addresses.length === 0) {
      addrHtml += `<p style="grid-column:1/-1;color:var(--text-muted);">Henüz kayıtlı adresiniz yok.</p>`;
    } else {
      addresses.forEach(a => {
        addrHtml += `
          <div class="card" style="padding:15px;background:#151821;border-radius:12px;border:1px solid var(--border);">
            <div style="font-weight:bold;margin-bottom:5px;">${a.title}</div>
            <div style="font-size:13px;color:var(--text-muted);margin-bottom:10px;">${a.city}, ${a.country}</div>
            <div style="font-size:13px;margin-bottom:10px;">${a.address_line1 || ''} ${a.address_line2 || ''}</div>
            <div style="display:flex;gap:10px;">
              <button class="btn btn-primary btn-sm prof-edit-addr" data-id="${a.id}"><i class="fas fa-edit"></i> Düzenle</button>
              <button class="btn btn-secondary btn-sm prof-del-addr" data-id="${a.id}"><i class="fas fa-trash"></i> Sil</button>
            </div>
          </div>
        `;
      });
    }
    addrHtml += '</div>';

    content.innerHTML = addrHtml;

    document.getElementById('prof-add-addr').addEventListener('click', () => {
      // Modalda form göster
      const modal = document.getElementById('modal-overlay');
      const modalContent = document.getElementById('modal-content');
      modalContent.innerHTML = `
        <h3>Yeni Adres Ekle</h3>
        <form id="prof-new-addr-form" style="margin-top:15px">
          <div class="form-group"><label>Adres Başlığı</label><input type="text" id="na-title" required></div>
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:10px;">
            <div class="form-group"><label>Ad</label><input type="text" id="na-first" required></div>
            <div class="form-group"><label>Soyad</label><input type="text" id="na-last" required></div>
          </div>
          <div class="form-group"><label>Şehir</label><input type="text" id="na-city" required></div>
          <div class="form-group"><label>Posta Kodu</label><input type="text" id="na-postal" required></div>
          <div class="form-group"><label>Açık Adres (Satır 1)</label><textarea id="na-line1" required rows="2"></textarea></div>
          <button type="submit" class="btn btn-primary" style="width:100%;justify-content:center">Kaydet</button>
        </form>
      `;
      modal.classList.remove('hidden');

      document.getElementById('prof-new-addr-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        try {
          await API.addresses.create({
            title: document.getElementById('na-title').value.trim(),
            first_name: document.getElementById('na-first').value.trim(),
            last_name: document.getElementById('na-last').value.trim(),
            city: document.getElementById('na-city').value.trim(),
            postal_code: document.getElementById('na-postal').value.trim(),
            address_line1: document.getElementById('na-line1').value.trim(),
            country: 'Türkiye'
          });
          closeModal();
          showToast('Adres eklendi', 'success');
          loadProfileAddresses();
        } catch (err) {
          showToast(err.message, 'error');
        }
      });
    });

    document.querySelectorAll('.prof-del-addr').forEach(btn => {
      btn.addEventListener('click', async () => {
        if(confirm('Silmek istediğinize emin misiniz?')) {
          try {
            await API.addresses.delete(btn.dataset.id);
            showToast('Adres silindi', 'success');
            loadProfileAddresses();
          } catch(err) {
            showToast(err.message, 'error');
          }
        }
      });
    });

    document.querySelectorAll('.prof-edit-addr').forEach(btn => {
      btn.addEventListener('click', () => {
        const id = parseInt(btn.dataset.id);
        const a = addresses.find(x => x.id === id);
        if(!a) return;
        
        const modal = document.getElementById('modal-overlay');
        const modalContent = document.getElementById('modal-content');
        modalContent.innerHTML = `
          <h3>Adresi Güncelle</h3>
          <form id="prof-edit-addr-form" style="margin-top:15px">
            <div class="form-group"><label>Adres Başlığı</label><input type="text" id="ea-title" value="${a.title}" required></div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:10px;">
              <div class="form-group"><label>Ad</label><input type="text" id="ea-first" value="${a.first_name || ''}" required></div>
              <div class="form-group"><label>Soyad</label><input type="text" id="ea-last" value="${a.last_name || ''}" required></div>
            </div>
            <div class="form-group"><label>Şehir</label><input type="text" id="ea-city" value="${a.city}" required></div>
            <div class="form-group"><label>Posta Kodu</label><input type="text" id="ea-postal" value="${a.postal_code || ''}" required></div>
            <div class="form-group"><label>Açık Adres (Satır 1)</label><textarea id="ea-line1" required rows="2">${a.address_line1 || a.address_line || ''}</textarea></div>
            <button type="submit" class="btn btn-primary" style="width:100%;justify-content:center">Güncelle</button>
          </form>
        `;
        modal.classList.remove('hidden');

        document.getElementById('prof-edit-addr-form').addEventListener('submit', async (e) => {
          e.preventDefault();
          try {
            await API.addresses.update(id, {
              title: document.getElementById('ea-title').value.trim(),
              first_name: document.getElementById('ea-first').value.trim(),
              last_name: document.getElementById('ea-last').value.trim(),
              city: document.getElementById('ea-city').value.trim(),
              postal_code: document.getElementById('ea-postal').value.trim(),
              address_line1: document.getElementById('ea-line1').value.trim(),
              country: 'Türkiye'
            });
            closeModal();
            showToast('Adres güncellendi', 'success');
            loadProfileAddresses();
          } catch(err) {
            showToast(err.message, 'error');
          }
        });
      });
    });

  } catch (err) {
    content.innerHTML = `<p style="color:var(--danger)">Hata: ${err.message}</p>`;
  }
}

async function loadProfileOrders() {
  const content = document.getElementById('profile-content');
  content.innerHTML = '<div class="spinner"></div>';

  try {
    const orders = await API.orders.list();
    let ordHtml = `
      <div style="margin-bottom:15px;">
        <h3 style="color:var(--text);font-size:18px;">Sipariş Geçmişim</h3>
      </div>
      <div style="display:flex;flex-direction:column;gap:15px;">
    `;

    if(orders.length === 0) {
      ordHtml += `<p style="color:var(--text-muted);">Henüz siparişiniz yok.</p>`;
    } else {
      orders.forEach(o => {
        ordHtml += `
          <div class="card" style="padding:15px;background:#151821;border-radius:12px;border:1px solid var(--border);display:flex;justify-content:space-between;align-items:center;">
            <div>
              <div style="font-weight:bold;margin-bottom:5px;">Sipariş #${o.order_number}</div>
              <div style="font-size:12px;color:var(--text-muted);">${formatDate(o.created_at)}</div>
            </div>
            <div>
              ${statusBadge(o.status)}
              <div style="font-weight:bold;margin-top:5px;text-align:right;">${formatPrice(o.total)}</div>
            </div>
            <button class="btn btn-secondary btn-sm" onclick="App.navigate('orders')">Detay</button>
          </div>
        `;
      });
    }
    ordHtml += '</div>';

    content.innerHTML = ordHtml;

  } catch (err) {
    content.innerHTML = `<p style="color:var(--danger)">Hata: ${err.message}</p>`;
  }
}
