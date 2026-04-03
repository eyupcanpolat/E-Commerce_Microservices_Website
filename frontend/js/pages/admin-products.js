/**
 * admin-products.js
 * Admin page for managing products (CRUD)
 */

async function renderAdminProductsPage() {
  const user = Auth.getUser();
  if (!user || user.role !== 'admin') {
    showToast('Bu sayfaya erişim yetkiniz yok.', 'error');
    App.navigate('home');
    return;
  }

  const container = document.getElementById('app-root');
  container.innerHTML = `
    <div style="padding:20px; max-width:1200px; margin:0 auto;">
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:20px;">
        <h2>Ürün Yönetimi</h2>
        <button class="btn btn-primary" id="btn-admin-add-product"><i class="fas fa-plus"></i> Yeni Ürün Ekle</button>
      </div>
      <div id="admin-products-container"><div class="spinner"></div></div>
    </div>
  `;

  document.getElementById('btn-admin-add-product').addEventListener('click', openAddProductModal);
  
  await loadAdminProducts();
}

async function loadAdminProducts() {
  const container = document.getElementById('admin-products-container');
  try {
    const res = await API.products.list({ page: 1 });
    // Assuming API.products.list returns paginated data inside res.data or just an array if small
    // The model PaginatedProducts has Data array.
    let products = Array.isArray(res) ? res : (res.data || []);
    
    if (products.length === 0) {
      container.innerHTML = '<p class="text-muted">Sistemde henüz ürün kayıtlı değil.</p>';
      return;
    }

    let html = `
      <div class="card" style="background:#151821; border-radius:12px; border:1px solid var(--border); overflow-x:auto;">
        <table style="width:100%; border-collapse: collapse; text-align:left;">
          <thead>
            <tr style="border-bottom: 1px solid var(--border);">
              <th style="padding:15px;">Görsel</th>
              <th style="padding:15px;">İsim</th>
              <th style="padding:15px;">Fiyat</th>
              <th style="padding:15px;">Stok</th>
              <th style="padding:15px;">Öne Çıkan</th>
              <th style="padding:15px; text-align:right;">İşlemler</th>
            </tr>
          </thead>
          <tbody>
    `;

    products.forEach(p => {
      html += `
        <tr style="border-bottom: 1px solid var(--border);">
          <td style="padding:15px;"><img src="${p.image_url || 'images/placeholder.png'}" style="width:50px; height:50px; border-radius:8px; object-fit:cover;"></td>
          <td style="padding:15px;">${p.name} <div style="font-size:11px; color:var(--text-muted)">SKU: ${p.sku}</div></td>
          <td style="padding:15px;">${formatPrice(p.sale_price > 0 ? p.sale_price : p.price)}</td>
          <td style="padding:15px;">${p.stock_quantity} (${p.stock_status})</td>
          <td style="padding:15px;">${p.is_featured ? '<i class="fas fa-check" style="color:var(--success)"></i>' : '-'}</td>
          <td style="padding:15px; text-align:right;">
            <button class="btn btn-secondary btn-sm admin-edit-pr" data-id="${p.id}" style="margin-right:5px;"><i class="fas fa-edit"></i> Güncelle</button>
            <button class="btn btn-secondary btn-sm admin-del-pr" data-id="${p.id}"><i class="fas fa-trash"></i> Sil</button>
          </td>
        </tr>
      `;
      // Ensure the button works by attaching raw product JSON to it via dataset or fetching it again
    });

    html += `</tbody></table></div>`;
    container.innerHTML = html;

    // Attach events
    document.querySelectorAll('.admin-del-pr').forEach(btn => {
      btn.addEventListener('click', async (e) => {
        if(!confirm('Bu ürünü silmek istediğinize emin misiniz?')) return;
        const id = e.currentTarget.dataset.id;
        try {
          await API.products.delete(id);
          showToast('Ürün başarıyla silindi', 'success');
          loadAdminProducts();
        } catch(err) {
          showToast(err.message, 'error');
        }
      });
    });

    document.querySelectorAll('.admin-edit-pr').forEach(btn => {
      btn.addEventListener('click', async (e) => {
        const id = e.currentTarget.dataset.id;
        // Fetch detailed product
        try {
          const detail = await API.products.get(id);
          openEditProductModal(detail);
        } catch(err) {
          showToast('Ürün detayları alınamadı', 'error');
        }
      });
    });

  } catch (err) {
    container.innerHTML = `<p style="color:var(--danger)">Hata: ${err.message}</p>`;
  }
}

function renderProductFormBody(p = {}) {
  // Common HTML string for both Add and Edit
  return `
    <div style="display:grid; grid-template-columns:1fr 1fr; gap:15px;">
      <div class="form-group"><label>Ürün Adı (*)</label><input type="text" id="pr-name" value="${p.name || ''}" required></div>
      <div class="form-group"><label>URL Slug (*)</label><input type="text" id="pr-slug" value="${p.slug || ''}" required></div>
    </div>
    
    <div style="display:grid; grid-template-columns:1fr 1fr; gap:15px;">
      <div class="form-group"><label>Normal Fiyat (*)</label><input type="number" step="0.01" id="pr-price" value="${p.price || ''}" required></div>
      <div class="form-group"><label>İndirimli Fiyat</label><input type="number" step="0.01" id="pr-saleprice" value="${p.sale_price || ''}"></div>
    </div>
    
    <div style="display:grid; grid-template-columns:1fr 1fr; gap:15px;">
      <div class="form-group"><label>Stok Miktarı</label><input type="number" id="pr-stock" value="${p.stock_quantity !== undefined ? p.stock_quantity : ''}"></div>
      <div class="form-group"><label>SKU (Stok Kodu)</label><input type="text" id="pr-sku" value="${p.sku || ''}"></div>
    </div>
    
    <div class="form-group"><label>Kısa Açıklama</label><input type="text" id="pr-shortdesc" value="${p.short_description || ''}"></div>
    
    <div class="form-group"><label>Ana Görsel URL</label><input type="text" id="pr-image" value="${p.image_url || ''}"></div>
    <div class="form-group">
      <label>Diğer Görseller (Virgülle ayırarak giriniz)</label>
      <input type="text" id="pr-gallery" placeholder="images/p1.jpg, images/p2.jpg" value="${(p.gallery_images || []).join(', ')}">
    </div>
    
    <div class="form-group"><label>Detaylı Açıklama</label><textarea id="pr-desc" rows="4">${p.description || ''}</textarea></div>
    
    <div class="form-group" style="display:flex; align-items:center;">
      <input type="checkbox" id="pr-featured" ${p.is_featured ? 'checked' : ''} style="margin-right:10px;">
      <label for="pr-featured" style="margin-bottom:0;">Öne Çıkan Ürün Yap</label>
    </div>
  `;
}

function extractProductFormData() {
  const galleryRaw = document.getElementById('pr-gallery').value;
  return {
    name: document.getElementById('pr-name').value.trim(),
    slug: document.getElementById('pr-slug').value.trim(),
    price: parseFloat(document.getElementById('pr-price').value),
    sale_price: document.getElementById('pr-saleprice').value ? parseFloat(document.getElementById('pr-saleprice').value) : null,
    stock_quantity: parseInt(document.getElementById('pr-stock').value) || 0,
    sku: document.getElementById('pr-sku').value.trim(),
    short_description: document.getElementById('pr-shortdesc').value.trim(),
    description: document.getElementById('pr-desc').value.trim(),
    image_url: document.getElementById('pr-image').value.trim(),
    gallery_images: galleryRaw ? galleryRaw.split(',').map(s => s.trim()).filter(s => s) : [],
    is_featured: document.getElementById('pr-featured').checked
  };
}

function openAddProductModal() {
  const modal = document.getElementById('modal-overlay');
  const modalContent = document.getElementById('modal-content');

  // To prevent the modal from getting stuck if it's too tall, add some scrolling.
  modalContent.style.maxHeight = '80vh';
  modalContent.style.overflowY = 'auto';

  modalContent.innerHTML = `
    <h3>Yeni Ürün Ekle</h3>
    <form id="admin-add-pr-form" style="margin-top:15px">
      ${renderProductFormBody()}
      <button type="submit" class="btn btn-primary" style="margin-top:20px; width:100%; justify-content:center;">Ürünü Kaydet</button>
    </form>
  `;
  modal.classList.remove('hidden');

  document.getElementById('admin-add-pr-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = extractProductFormData();
    try {
      await API.products.create(data);
      closeModal();
      showToast('Ürün başarıyla oluşturuldu!', 'success');
      loadAdminProducts();
    } catch(err) {
      showToast(err.message, 'error');
    }
  });
}

function openEditProductModal(product) {
  const modal = document.getElementById('modal-overlay');
  const modalContent = document.getElementById('modal-content');
  
  modalContent.style.maxHeight = '80vh';
  modalContent.style.overflowY = 'auto';

  modalContent.innerHTML = `
    <h3>Ürün Güncelle</h3>
    <form id="admin-edit-pr-form" style="margin-top:15px">
      ${renderProductFormBody(product)}
      <button type="submit" class="btn btn-primary" style="margin-top:20px; width:100%; justify-content:center;">Değişiklikleri Kaydet</button>
    </form>
  `;
  modal.classList.remove('hidden');

  document.getElementById('admin-edit-pr-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = extractProductFormData();
    try {
      await API.products.update(product.id, data);
      closeModal();
      showToast('Ürün başarıyla güncellendi!', 'success');
      loadAdminProducts();
    } catch(err) {
      showToast(err.message, 'error');
    }
  });
}
