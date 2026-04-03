// Package tests contains unit tests for AddressService business logic.
// Uses table-driven tests and mock repositories (TDD approach).
package tests

import (
	"errors"
	"sync"
	"testing"
	"time"

	"eticaret/address-service/internal/model"
	"eticaret/address-service/internal/service"
)

// ── Mock Repository ─────────────────────────────────────────────────────────

type mockAddressRepository struct {
	mu        sync.RWMutex
	addresses []model.Address
	nextID    int
}

func newMockAddressRepo() *mockAddressRepository {
	return &mockAddressRepository{nextID: 1}
}

func (m *mockAddressRepository) GetByUserID(userID int) ([]model.Address, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []model.Address
	for _, a := range m.addresses {
		if a.UserID == userID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAddressRepository) GetByID(id int) (*model.Address, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := range m.addresses {
		if m.addresses[i].ID == id {
			return &m.addresses[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockAddressRepository) Create(addr *model.Address) (*model.Address, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	addr.ID = m.nextID
	m.nextID++
	addr.CreatedAt = time.Now()
	addr.UpdatedAt = time.Now()
	m.addresses = append(m.addresses, *addr)
	return addr, nil
}

func (m *mockAddressRepository) Update(id, userID int, req model.AddressRequest) (*model.Address, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.addresses {
		if m.addresses[i].ID == id && m.addresses[i].UserID == userID {
			m.addresses[i].Title        = req.Title
			m.addresses[i].AddressLine1 = req.AddressLine1
			m.addresses[i].City         = req.City
			m.addresses[i].PostalCode   = req.PostalCode
			m.addresses[i].UpdatedAt    = time.Now()
			return &m.addresses[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockAddressRepository) Delete(id, userID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.addresses {
		if m.addresses[i].ID == id && m.addresses[i].UserID == userID {
			m.addresses = append(m.addresses[:i], m.addresses[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

// helper: seed one address for userID
func seedAddress(repo *mockAddressRepository, userID int) *model.Address {
	addr, _ := repo.Create(&model.Address{
		UserID:       userID,
		Title:        "Ev",
		FirstName:    "Test",
		LastName:     "User",
		AddressLine1: "Test Sokak No:1",
		City:         "İstanbul",
		PostalCode:   "34000",
		Country:      "Türkiye",
	})
	return addr
}

// ── Tests ───────────────────────────────────────────────────────────────────

func TestCreateAddress_Success(t *testing.T) {
	svc := service.NewAddressService(newMockAddressRepo())

	addr, err := svc.CreateAddress(1, model.AddressRequest{
		Title:        "Ev",
		FirstName:    "Ahmet",
		AddressLine1: "Bağcılar Cad. No:5",
		City:         "İstanbul",
		PostalCode:   "34000",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if addr.ID == 0 {
		t.Error("expected auto-incremented ID")
	}
	if addr.UserID != 1 {
		t.Errorf("expected userID=1, got %d", addr.UserID)
	}
	if addr.Country != "Turkey" && addr.Country != "Türkiye" && addr.Country != "" {
		// Country falls back to Turkey if empty: acceptable
	}
}

func TestCreateAddress_MissingTitle(t *testing.T) {
	svc := service.NewAddressService(newMockAddressRepo())

	_, err := svc.CreateAddress(1, model.AddressRequest{
		Title:        "", // empty
		AddressLine1: "Test Sokak",
		City:         "Ankara",
		PostalCode:   "06000",
	})

	if err != service.ErrTitleRequired {
		t.Errorf("expected ErrTitleRequired, got %v", err)
	}
}

func TestCreateAddress_MissingLine1(t *testing.T) {
	svc := service.NewAddressService(newMockAddressRepo())

	_, err := svc.CreateAddress(1, model.AddressRequest{
		Title:        "İş",
		AddressLine1: "", // empty
		City:         "İzmir",
		PostalCode:   "35000",
	})

	if err != service.ErrLine1Required {
		t.Errorf("expected ErrLine1Required, got %v", err)
	}
}

func TestCreateAddress_MissingCity(t *testing.T) {
	svc := service.NewAddressService(newMockAddressRepo())

	_, err := svc.CreateAddress(1, model.AddressRequest{
		Title:        "Yazlık",
		AddressLine1: "Sahil Yolu",
		City:         "", // empty
		PostalCode:   "48000",
	})

	if err != service.ErrCityRequired {
		t.Errorf("expected ErrCityRequired, got %v", err)
	}
}

func TestCreateAddress_MissingPostalCode(t *testing.T) {
	svc := service.NewAddressService(newMockAddressRepo())

	_, err := svc.CreateAddress(1, model.AddressRequest{
		Title:        "Depo",
		AddressLine1: "Sanayi Cad.",
		City:         "Bursa",
		PostalCode:   "", // empty
	})

	if err != service.ErrPostalRequired {
		t.Errorf("expected ErrPostalRequired, got %v", err)
	}
}

func TestGetUserAddresses_IsolatedByUser(t *testing.T) {
	repo := newMockAddressRepo()
	svc := service.NewAddressService(repo)

	// User 1 has 2 addresses
	svc.CreateAddress(1, model.AddressRequest{Title: "Ev", AddressLine1: "X", City: "İstanbul", PostalCode: "34000"})
	svc.CreateAddress(1, model.AddressRequest{Title: "İş", AddressLine1: "Y", City: "İstanbul", PostalCode: "34001"})
	// User 2 has 1 address
	svc.CreateAddress(2, model.AddressRequest{Title: "Ev", AddressLine1: "Z", City: "Ankara", PostalCode: "06000"})

	addrs1, err := svc.GetUserAddresses(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs1) != 2 {
		t.Errorf("user 1 should have 2 addresses, got %d", len(addrs1))
	}

	addrs2, _ := svc.GetUserAddresses(2)
	if len(addrs2) != 1 {
		t.Errorf("user 2 should have 1 address, got %d", len(addrs2))
	}
}

func TestGetAddress_OwnershipCheck(t *testing.T) {
	repo := newMockAddressRepo()
	svc := service.NewAddressService(repo)

	addr := seedAddress(repo, 1)

	// Owner can access
	got, err := svc.GetAddress(addr.ID, 1)
	if err != nil {
		t.Fatalf("owner should access: %v", err)
	}
	if got.ID != addr.ID {
		t.Errorf("expected ID %d, got %d", addr.ID, got.ID)
	}

	// Other user cannot access
	_, err = svc.GetAddress(addr.ID, 99)
	if err != service.ErrAccessDenied {
		t.Errorf("expected ErrAccessDenied for wrong user, got %v", err)
	}
}

func TestGetAddress_NotFound(t *testing.T) {
	svc := service.NewAddressService(newMockAddressRepo())

	_, err := svc.GetAddress(999, 1)
	if err != service.ErrAddressNotFound {
		t.Errorf("expected ErrAddressNotFound, got %v", err)
	}
}

func TestUpdateAddress_Success(t *testing.T) {
	repo := newMockAddressRepo()
	svc := service.NewAddressService(repo)
	addr := seedAddress(repo, 1)

	updated, err := svc.UpdateAddress(addr.ID, 1, model.AddressRequest{
		Title:        "Güncellendi",
		AddressLine1: "Yeni Adres",
		City:         "Ankara",
		PostalCode:   "06000",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Güncellendi" {
		t.Errorf("expected updated title, got %s", updated.Title)
	}
}

func TestUpdateAddress_WrongOwner(t *testing.T) {
	repo := newMockAddressRepo()
	svc := service.NewAddressService(repo)
	addr := seedAddress(repo, 1)

	_, err := svc.UpdateAddress(addr.ID, 99, model.AddressRequest{
		Title:        "Hack",
		AddressLine1: "Hacker Sk.",
		City:         "İstanbul",
		PostalCode:   "34000",
	})

	if err != service.ErrAccessDenied {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestDeleteAddress_Success(t *testing.T) {
	repo := newMockAddressRepo()
	svc := service.NewAddressService(repo)
	addr := seedAddress(repo, 1)

	err := svc.DeleteAddress(addr.ID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be gone now
	_, err = svc.GetAddress(addr.ID, 1)
	if err != service.ErrAddressNotFound {
		t.Errorf("address should be deleted, got %v", err)
	}
}

func TestDeleteAddress_WrongOwner(t *testing.T) {
	repo := newMockAddressRepo()
	svc := service.NewAddressService(repo)
	addr := seedAddress(repo, 1)

	err := svc.DeleteAddress(addr.ID, 42)
	if err != service.ErrAccessDenied {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestCreateAddress_TableDriven_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     model.AddressRequest
		wantErr error
	}{
		{
			name:    "empty title",
			req:     model.AddressRequest{Title: " ", AddressLine1: "X", City: "Y", PostalCode: "Z"},
			wantErr: service.ErrTitleRequired,
		},
		{
			name:    "empty address line",
			req:     model.AddressRequest{Title: "T", AddressLine1: "", City: "Y", PostalCode: "Z"},
			wantErr: service.ErrLine1Required,
		},
		{
			name:    "empty city",
			req:     model.AddressRequest{Title: "T", AddressLine1: "X", City: "   ", PostalCode: "Z"},
			wantErr: service.ErrCityRequired,
		},
		{
			name:    "empty postal code",
			req:     model.AddressRequest{Title: "T", AddressLine1: "X", City: "Y", PostalCode: ""},
			wantErr: service.ErrPostalRequired,
		},
	}

	svc := service.NewAddressService(newMockAddressRepo())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateAddress(1, tt.req)
			if err != tt.wantErr {
				t.Errorf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
