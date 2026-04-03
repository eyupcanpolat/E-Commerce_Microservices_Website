// Package handler contains HTTP handlers for AuthService endpoints.
// Note: JWT validation is done at the API Gateway level.
// This service trusts X-User-* headers injected by the gateway.
package handler

import (
	"encoding/json"
	"net/http"

	"eticaret/auth-service/internal/middleware"
	"eticaret/auth-service/internal/model"
	"eticaret/auth-service/internal/service"
	"eticaret/shared/logger"
	"eticaret/shared/response"
)

// AuthHandler handles HTTP requests for authentication.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{authService: svc}
}

// Register handles POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Geçersiz JSON formatı")
		return
	}

	result, err := h.authService.Register(req)
	if err != nil {
		logger.Warn("Register failed", "email", req.Email, "error", err.Error())
		if err == service.ErrEmailExists {
			response.Conflict(w, err.Error())
			return
		}
		response.BadRequest(w, err.Error())
		return
	}

	logger.Info("User registered", "email", req.Email)
	response.Created(w, "Kayıt başarılı", result)
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Geçersiz JSON formatı")
		return
	}

	result, err := h.authService.Login(req)
	if err != nil {
		logger.Warn("Login failed", "email", req.Email, "error", err.Error())
		response.Unauthorized(w, err.Error())
		return
	}

	logger.Info("User logged in", "email", req.Email)
	response.Success(w, "Giriş başarılı", result)
}

// Health handles GET /health
func (h *AuthHandler) Health(w http.ResponseWriter, r *http.Request) {
	// Read user ID from gateway-injected header (if authenticated request)
	userID := middleware.GetUserID(r)
	info := map[string]interface{}{
		"service": "auth-service",
		"status":  "ok",
		"network_isolation": "active",
	}
	if userID > 0 {
		info["authenticated_as"] = userID
	}
	response.Success(w, "auth-service is healthy", info)
}

// UpdateProfile handles PUT /auth/profile
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == 0 {
		response.Unauthorized(w, "Oturum süresi dolmuş veya geçersiz")
		return
	}

	var req model.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Geçersiz JSON formatı")
		return
	}

	updatedUser, err := h.authService.UpdateProfile(userID, req)
	if err != nil {
		logger.Warn("Profile update failed", "user_id", userID, "error", err.Error())
		if err == service.ErrPasswordTooShort {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalServerError(w, err.Error())
		return
	}

	logger.Info("Profile updated", "user_id", userID)
	response.Success(w, "Profil başarıyla güncellendi", updatedUser)
}
