// Package handler - order handlers using gateway-injected identity headers.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"eticaret/order-service/internal/middleware"
	"eticaret/order-service/internal/model"
	"eticaret/order-service/internal/service"
	"eticaret/shared/logger"
	"eticaret/shared/response"
)

type OrderHandler struct {
	orderService service.OrderService
}

func NewOrderHandler(svc service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: svc}
}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	orders, err := h.orderService.GetUserOrders(userID)
	if err != nil {
		response.InternalServerError(w, "")
		return
	}
	response.Success(w, "", orders)
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Geçersiz sipariş ID")
		return
	}
	order, err := h.orderService.GetOrder(id, userID)
	if err != nil {
		if err == service.ErrAccessDenied {
			response.Forbidden(w, err.Error())
			return
		}
		response.NotFound(w, err.Error())
		return
	}
	response.Success(w, "", order)
}

func (h *OrderHandler) GetOrderByNumber(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	orderNumber := r.PathValue("orderNumber")
	order, err := h.orderService.GetOrderByNumber(orderNumber, userID)
	if err != nil {
		if err == service.ErrAccessDenied {
			response.Forbidden(w, err.Error())
			return
		}
		response.NotFound(w, err.Error())
		return
	}
	response.Success(w, "", order)
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	var req model.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Geçersiz JSON")
		return
	}
	order, err := h.orderService.CreateOrder(userID, req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	logger.Info("Order created", "order_number", order.OrderNumber, "user_id", userID, "total", order.Total)
	response.Created(w, "Sipariş oluşturuldu", order)
}

func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	orderNumber := r.PathValue("orderNumber")
	if err := h.orderService.CancelOrder(orderNumber, userID); err != nil {
		if err == service.ErrAccessDenied {
			response.Forbidden(w, err.Error())
			return
		}
		response.BadRequest(w, err.Error())
		return
	}
	response.Success(w, "Sipariş iptal edildi", nil)
}

func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Geçersiz sipariş ID")
		return
	}
	var req model.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Geçersiz JSON")
		return
	}
	order, err := h.orderService.UpdateStatus(id, req.Status)
	if err != nil {
		response.NotFound(w, err.Error())
		return
	}
	response.Success(w, "Sipariş durumu güncellendi", order)
}

func (h *OrderHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.Success(w, "order-service is healthy", map[string]string{
		"service":           "order-service",
		"status":            "ok",
		"network_isolation": "active",
	})
}
