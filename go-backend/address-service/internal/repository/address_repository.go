// Package repository handles data access for AddressService.
package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"eticaret/address-service/internal/model"
)

// AddressRepository defines the interface for address data operations.
type AddressRepository interface {
	GetByUserID(userID int) ([]model.Address, error)
	GetByID(id int) (*model.Address, error)
	Create(addr *model.Address) (*model.Address, error)
	Update(id, userID int, req model.AddressRequest) (*model.Address, error)
	Delete(id, userID int) error
}

type jsonAddressRepository struct {
	filePath string
	mu       sync.RWMutex
}

func NewAddressRepository(filePath string) AddressRepository {
	return &jsonAddressRepository{filePath: filePath}
}

func (r *jsonAddressRepository) load() ([]model.Address, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read addresses file: %w", err)
	}
	var addresses []model.Address
	return addresses, json.Unmarshal(data, &addresses)
}

func (r *jsonAddressRepository) save(addresses []model.Address) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.MarshalIndent(addresses, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0644)
}

func (r *jsonAddressRepository) GetByUserID(userID int) ([]model.Address, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	var result []model.Address
	for _, addr := range all {
		if addr.UserID == userID {
			result = append(result, addr)
		}
	}
	return result, nil
}

func (r *jsonAddressRepository) GetByID(id int) (*model.Address, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, errors.New("address not found")
}

func (r *jsonAddressRepository) Create(addr *model.Address) (*model.Address, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}

	// Auto-increment ID
	maxID := 0
	for _, a := range all {
		if a.ID > maxID {
			maxID = a.ID
		}
	}
	addr.ID = maxID + 1
	addr.CreatedAt = time.Now()
	addr.UpdatedAt = time.Now()

	// If this is default, unset other defaults for same user
	if addr.IsDefault {
		for i := range all {
			if all[i].UserID == addr.UserID {
				all[i].IsDefault = false
			}
		}
	}

	all = append(all, *addr)
	return addr, r.save(all)
}

func (r *jsonAddressRepository) Update(id, userID int, req model.AddressRequest) (*model.Address, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}

	// If updating to default, unset others first
	if req.IsDefault {
		for i := range all {
			if all[i].UserID == userID && all[i].ID != id {
				all[i].IsDefault = false
			}
		}
	}

	for i := range all {
		if all[i].ID == id && all[i].UserID == userID {
			all[i].Title = req.Title
			all[i].FirstName = req.FirstName
			all[i].LastName = req.LastName
			all[i].Phone = req.Phone
			all[i].AddressLine1 = req.AddressLine1
			all[i].AddressLine2 = req.AddressLine2
			all[i].City = req.City
			all[i].State = req.State
			all[i].PostalCode = req.PostalCode
			all[i].Country = req.Country
			all[i].IsDefault = req.IsDefault
			all[i].UpdatedAt = time.Now()
			if err := r.save(all); err != nil {
				return nil, err
			}
			return &all[i], nil
		}
	}
	return nil, errors.New("address not found or access denied")
}

func (r *jsonAddressRepository) Delete(id, userID int) error {
	all, err := r.load()
	if err != nil {
		return err
	}
	for i := range all {
		if all[i].ID == id && all[i].UserID == userID {
			all = append(all[:i], all[i+1:]...)
			return r.save(all)
		}
	}
	return errors.New("address not found or access denied")
}
