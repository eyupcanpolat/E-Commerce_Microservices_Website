// Package response provides standardized HTTP JSON responses
// used consistently across all microservices.
package response

import (
	"encoding/json"
	"net/http"
)

// APIResponse is the standard response envelope for all API endpoints.
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// JSON writes a JSON response with the given status code and payload.
func JSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// Success sends a 200 OK response with data.
func Success(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Created sends a 201 Created response with data.
func Created(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// BadRequest sends a 400 error response.
func BadRequest(w http.ResponseWriter, message string) {
	JSON(w, http.StatusBadRequest, APIResponse{
		Success: false,
		Error:   message,
	})
}

// Unauthorized sends a 401 error. Used when JWT is missing or invalid.
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "unauthorized: invalid or missing token"
	}
	JSON(w, http.StatusUnauthorized, APIResponse{
		Success: false,
		Error:   message,
	})
}

// Forbidden sends a 403 error. Used when user lacks permission (e.g. non-admin).
func Forbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "forbidden: insufficient permissions"
	}
	JSON(w, http.StatusForbidden, APIResponse{
		Success: false,
		Error:   message,
	})
}

// NotFound sends a 404 error.
func NotFound(w http.ResponseWriter, message string) {
	JSON(w, http.StatusNotFound, APIResponse{
		Success: false,
		Error:   message,
	})
}

// InternalServerError sends a 500 error.
func InternalServerError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "internal server error"
	}
	JSON(w, http.StatusInternalServerError, APIResponse{
		Success: false,
		Error:   message,
	})
}

// Conflict sends a 409 error. Used when resource already exists.
func Conflict(w http.ResponseWriter, message string) {
	JSON(w, http.StatusConflict, APIResponse{
		Success: false,
		Error:   message,
	})
}
