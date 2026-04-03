// Package handler - address handlers using gateway-injected identity headers.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"eticaret/address-service/internal/middleware"
	"eticaret/address-service/internal/model"
	"eticaret/address-service/internal/service"
	"eticaret/shared/logger"
	"eticaret/shared/response"
)

type AddressHandler struct {
	addressService service.AddressService
}

func NewAddressHandler(svc service.AddressService) *AddressHandler {
	return &AddressHandler{addressService: svc}
}

func (h *AddressHandler) GetAddresses(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	addresses, err := h.addressService.GetUserAddresses(userID)
	if err != nil {
		response.InternalServerError(w, "")
		return
	}
	if addresses == nil {
		addresses = []model.Address{}
	}
	response.Success(w, "", addresses)
}

func (h *AddressHandler) GetAddress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Geçersiz adres ID")
		return
	}

	addr, err := h.addressService.GetAddress(id, userID)
	if err != nil {
		if err == service.ErrAccessDenied {
			response.Forbidden(w, err.Error())
			return
		}
		response.NotFound(w, err.Error())
		return
	}
	response.Success(w, "", addr)
}

func (h *AddressHandler) CreateAddress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req model.AddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Geçersiz JSON")
		return
	}

	addr, err := h.addressService.CreateAddress(userID, req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	logger.Info("Address created", "id", addr.ID, "user_id", userID)
	response.Created(w, "Adres eklendi", addr)
}

func (h *AddressHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Geçersiz adres ID")
		return
	}

	var req model.AddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Geçersiz JSON")
		return
	}

	addr, err := h.addressService.UpdateAddress(id, userID, req)
	if err != nil {
		if err == service.ErrAccessDenied {
			response.Forbidden(w, err.Error())
			return
		}
		response.BadRequest(w, err.Error())
		return
	}
	response.Success(w, "Adres güncellendi", addr)
}

func (h *AddressHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Geçersiz adres ID")
		return
	}

	if err := h.addressService.DeleteAddress(id, userID); err != nil {
		if err == service.ErrAccessDenied {
			response.Forbidden(w, err.Error())
			return
		}
		response.NotFound(w, err.Error())
		return
	}
	response.Success(w, "Adres silindi", nil)
}

func (h *AddressHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.Success(w, "address-service is healthy", map[string]string{
		"service":           "address-service",
		"status":            "ok",
		"network_isolation": "active",
	})
}
