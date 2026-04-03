// Package service contains business logic for AddressService.
package service

import (
	"errors"
	"strings"

	"eticaret/address-service/internal/model"
	"eticaret/address-service/internal/repository"
)

var (
	ErrAddressNotFound  = errors.New("adres bulunamadı")
	ErrTitleRequired    = errors.New("adres başlığı zorunludur")
	ErrLine1Required    = errors.New("adres satırı zorunludur")
	ErrCityRequired     = errors.New("şehir zorunludur")
	ErrPostalRequired   = errors.New("posta kodu zorunludur")
	ErrAccessDenied     = errors.New("bu adrese erişim yetkiniz yok")
)

// AddressService defines all address-related operations.
type AddressService interface {
	GetUserAddresses(userID int) ([]model.Address, error)
	GetAddress(id, userID int) (*model.Address, error)
	CreateAddress(userID int, req model.AddressRequest) (*model.Address, error)
	UpdateAddress(id, userID int, req model.AddressRequest) (*model.Address, error)
	DeleteAddress(id, userID int) error
}

type addressService struct {
	repo repository.AddressRepository
}

func NewAddressService(repo repository.AddressRepository) AddressService {
	return &addressService{repo: repo}
}

func (s *addressService) GetUserAddresses(userID int) ([]model.Address, error) {
	return s.repo.GetByUserID(userID)
}

func (s *addressService) GetAddress(id, userID int) (*model.Address, error) {
	addr, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrAddressNotFound
	}
	// Ownership check — user can only see their own addresses
	if addr.UserID != userID {
		return nil, ErrAccessDenied
	}
	return addr, nil
}

func (s *addressService) CreateAddress(userID int, req model.AddressRequest) (*model.Address, error) {
	if err := validateAddressRequest(req); err != nil {
		return nil, err
	}

	addr := &model.Address{
		UserID:       userID,
		Title:        strings.TrimSpace(req.Title),
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		Phone:        strings.TrimSpace(req.Phone),
		AddressLine1: strings.TrimSpace(req.AddressLine1),
		AddressLine2: strings.TrimSpace(req.AddressLine2),
		City:         strings.TrimSpace(req.City),
		State:        strings.TrimSpace(req.State),
		PostalCode:   strings.TrimSpace(req.PostalCode),
		Country:      req.Country,
		IsDefault:    req.IsDefault,
	}
	if addr.Country == "" {
		addr.Country = "Turkey"
	}

	return s.repo.Create(addr)
}

func (s *addressService) UpdateAddress(id, userID int, req model.AddressRequest) (*model.Address, error) {
	// Verify ownership before update
	if _, err := s.GetAddress(id, userID); err != nil {
		return nil, err
	}
	if err := validateAddressRequest(req); err != nil {
		return nil, err
	}
	return s.repo.Update(id, userID, req)
}

func (s *addressService) DeleteAddress(id, userID int) error {
	// Verify ownership before delete
	if _, err := s.GetAddress(id, userID); err != nil {
		return err
	}
	return s.repo.Delete(id, userID)
}

func validateAddressRequest(req model.AddressRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return ErrTitleRequired
	}
	if strings.TrimSpace(req.AddressLine1) == "" {
		return ErrLine1Required
	}
	if strings.TrimSpace(req.City) == "" {
		return ErrCityRequired
	}
	if strings.TrimSpace(req.PostalCode) == "" {
		return ErrPostalRequired
	}
	return nil
}
