/**
 * addresses.js — Kullanıcı Adres Yönetimi
 * JWT zorunlu. Kullanıcı sadece kendi adreslerini görebilir/düzenleyebilir.
 */

async function renderAddressesPage() {
  if (!Auth.isLoggedIn()) {
    showToast('Adreslerinizi görmek için giriş yapmalısınız', 'warning');
    App.navigate('login');
    return;
  }

  const root = document.getElementById('app-root');
  root.innerHTML = `
    <div class="container">
      <div class="section-header">
        <h1 class="section-title">Adreslerim</h1>
        <button class="btn btn-primary btn-sm" id="add-address-btn">
          <i class="fas fa-plus"></i> Yeni Adres Ekle
        </button>
      </div>
      <div id="addresses-grid" class="addresses-grid">
        <div class="page-loading"><div class="spinner"></div></div>
      </div>
    </div>`;

  document.getElementById('add-address-btn').addEventListener('click', () => openAddressModal());

  await loadAddresses();
}

async function loadAddresses() {
  const grid = document.getElementById('addresses-grid');
  grid.innerHTML = '<div class="page-loading"><div class="spinner"></div></div>';

  try {
    const addresses = await API.addresses.list();

    if (!addresses || addresses.length === 0) {
      grid.innerHTML = `
        <div class="no-addresses" style="grid-column:1/-1">
          <i class="fas fa-map-marker-alt"></i>
          <h2 style="margin-bottom:8px">Henüz adres eklemediniz</h2>
          <p>Teslimat adresi ekleyin ve daha hızlı sipariş verin.</p>
          <button class="btn btn-primary" style="margin-top:16px" onclick="openAddressModal()">
            <i class="fas fa-plus"></i> Adres Ekle
          </button>
        </div>`;
      return;
    }

    // Varsayılan adres önce
    const sorted = [...addresses].sort((a, b) => b.is_default - a.is_default);
    grid.innerHTML = sorted.map(a => addressCardHTML(a)).join('');

    // Bind events
    grid.querySelectorAll('.edit-address-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        const id = parseInt(btn.dataset.id);
        const addr = sorted.find(a => a.id === id);
        if (addr) openAddressModal(addr);
      });
    });

    grid.querySelectorAll('.delete-address-btn').forEach(btn => {
      btn.addEventListener('click', async () => {
        const id = parseInt(btn.dataset.id);
        if (!confirm('Bu adresi silmek istediğinizden emin misiniz?')) return;
        btn.disabled = true;
        try {
          await API.addresses.delete(id);
          showToast('Adres silindi', 'success');
          loadAddresses();
        } catch (err) {
          showToast(err.message, 'error');
          btn.disabled = false;
        }
      });
    });

  } catch (err) {
    grid.innerHTML = `
      <div class="no-addresses" style="grid-column:1/-1">
        <i class="fas fa-exclamation-triangle" style="color:var(--danger)"></i>
        <h2>Adresler yüklenemedi</h2>
        <p>${err.message}</p>
        <button class="btn btn-primary" onclick="loadAddresses()">Tekrar Dene</button>
      </div>`;
  }
}

function addressCardHTML(addr) {
  return `
    <div class="address-card ${addr.is_default ? 'is-default' : ''}">
      <div class="address-title">
        <i class="fas fa-map-marker-alt" style="color:var(--accent)"></i>
        ${addr.title}
        ${addr.is_default ? '<span class="default-tag">Varsayılan</span>' : ''}
      </div>
      <div class="address-line">
        <i class="fas fa-user" style="width:16px;color:var(--text-muted)"></i>
        ${addr.first_name} ${addr.last_name}
      </div>
      ${addr.phone ? `<div class="address-line">
        <i class="fas fa-phone" style="width:16px;color:var(--text-muted)"></i>
        ${addr.phone}
      </div>` : ''}
      <div class="address-line">
        <i class="fas fa-home" style="width:16px;color:var(--text-muted)"></i>
        ${addr.address_line1}
        ${addr.address_line2 ? ', ' + addr.address_line2 : ''}
      </div>
      <div class="address-line">
        <i class="fas fa-city" style="width:16px;color:var(--text-muted)"></i>
        ${addr.city}${addr.state ? ', ' + addr.state : ''} ${addr.postal_code}
      </div>
      <div class="address-line">
        <i class="fas fa-flag" style="width:16px;color:var(--text-muted)"></i>
        ${addr.country || 'Türkiye'}
      </div>
      <div class="address-actions">
        <button class="btn btn-secondary btn-sm edit-address-btn" data-id="${addr.id}">
          <i class="fas fa-edit"></i> Düzenle
        </button>
        <button class="btn btn-danger btn-sm delete-address-btn" data-id="${addr.id}">
          <i class="fas fa-trash"></i> Sil
        </button>
      </div>
    </div>`;
}

// ── Adres Modal (Ekle / Düzenle) ────────────────────────────────────────────
function openAddressModal(addr = null) {
  const isEdit = !!addr;
  const overlay = document.getElementById('modal-overlay');
  const content = document.getElementById('modal-content');

  content.innerHTML = `
    <h3 class="modal-title">
      <i class="fas fa-map-marker-alt" style="color:var(--accent)"></i>
      ${isEdit ? 'Adresi Düzenle' : 'Yeni Adres Ekle'}
    </h3>
    <div class="form-error" id="addr-error"></div>
    <form id="addr-form" novalidate>
      <div class="form-group">
        <label>Adres Başlığı <span style="color:var(--danger)">*</span></label>
        <input type="text" id="addr-title" placeholder="Ev, İş, vs." value="${isEdit ? addr.title : ''}">
      </div>
      <div class="form-grid-2">
        <div class="form-group">
          <label>Ad <span style="color:var(--danger)">*</span></label>
          <input type="text" id="addr-firstname" placeholder="Adınız" value="${isEdit ? addr.first_name : ''}">
        </div>
        <div class="form-group">
          <label>Soyad</label>
          <input type="text" id="addr-lastname" placeholder="Soyadınız" value="${isEdit ? addr.last_name : ''}">
        </div>
      </div>
      <div class="form-group">
        <label>Telefon</label>
        <input type="tel" id="addr-phone" placeholder="05XX XXX XXXX" value="${isEdit ? addr.phone : ''}">
      </div>
      <div class="form-group">
        <label>Adres Satırı 1 <span style="color:var(--danger)">*</span></label>
        <input type="text" id="addr-line1" placeholder="Sokak, cadde, bina no" value="${isEdit ? addr.address_line1 : ''}">
      </div>
      <div class="form-group">
        <label>Adres Satırı 2</label>
        <input type="text" id="addr-line2" placeholder="Daire, kat, blok (isteğe bağlı)" value="${isEdit ? addr.address_line2 || '' : ''}">
      </div>
      <div class="form-grid-2">
        <div class="form-group">
          <label>Şehir <span style="color:var(--danger)">*</span></label>
          <input type="text" id="addr-city" placeholder="İstanbul" value="${isEdit ? addr.city : ''}">
        </div>
        <div class="form-group">
          <label>İlçe/Eyalet</label>
          <input type="text" id="addr-state" placeholder="Kadıköy" value="${isEdit ? addr.state || '' : ''}">
        </div>
      </div>
      <div class="form-grid-2">
        <div class="form-group">
          <label>Posta Kodu <span style="color:var(--danger)">*</span></label>
          <input type="text" id="addr-postal" placeholder="34XXX" value="${isEdit ? addr.postal_code : ''}">
        </div>
        <div class="form-group">
          <label>Ülke</label>
          <input type="text" id="addr-country" placeholder="Türkiye" value="${isEdit ? addr.country || 'Türkiye' : 'Türkiye'}">
        </div>
      </div>
      <div class="form-group" style="display:flex;align-items:center;gap:10px">
        <input type="checkbox" id="addr-default" style="width:auto" ${isEdit && addr.is_default ? 'checked' : ''}>
        <label for="addr-default" style="margin:0;cursor:pointer">Varsayılan adres olarak ayarla</label>
      </div>
      <button type="submit" class="btn btn-primary" style="width:100%;justify-content:center;margin-top:8px" id="addr-save-btn">
        <i class="fas fa-save"></i> ${isEdit ? 'Değişiklikleri Kaydet' : 'Adresi Ekle'}
      </button>
    </form>`;

  overlay.classList.remove('hidden');

  document.getElementById('addr-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const errEl = document.getElementById('addr-error');
    const btn = document.getElementById('addr-save-btn');
    errEl.classList.remove('show');

    const payload = {
      title:         document.getElementById('addr-title').value.trim(),
      first_name:    document.getElementById('addr-firstname').value.trim(),
      last_name:     document.getElementById('addr-lastname').value.trim(),
      phone:         document.getElementById('addr-phone').value.trim(),
      address_line1: document.getElementById('addr-line1').value.trim(),
      address_line2: document.getElementById('addr-line2').value.trim(),
      city:          document.getElementById('addr-city').value.trim(),
      state:         document.getElementById('addr-state').value.trim(),
      postal_code:   document.getElementById('addr-postal').value.trim(),
      country:       document.getElementById('addr-country').value.trim() || 'Türkiye',
      is_default:    document.getElementById('addr-default').checked
    };

    if (!payload.title || !payload.first_name || !payload.address_line1 || !payload.city || !payload.postal_code) {
      errEl.textContent = 'Lütfen zorunlu alanları doldurun.';
      errEl.classList.add('show');
      return;
    }

    btn.disabled = true;
    btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Kaydediliyor...';

    try {
      if (isEdit) {
        await API.addresses.update(addr.id, payload);
        showToast('Adres güncellendi', 'success');
      } else {
        await API.addresses.create(payload);
        showToast('Adres eklendi', 'success');
      }
      closeModal();
      loadAddresses();
    } catch (err) {
      errEl.textContent = err.message || 'İşlem başarısız';
      errEl.classList.add('show');
      btn.disabled = false;
      btn.innerHTML = `<i class="fas fa-save"></i> ${isEdit ? 'Kaydet' : 'Ekle'}`;
    }
  });
}
