// Package service contains the business logic for AuthService.
// It sits between the HTTP handler and the repository layer.
package service

import (
	"errors"
	"regexp"
	"strings"

	"eticaret/auth-service/internal/model"
	"eticaret/auth-service/internal/repository"
	sharedJWT "eticaret/shared/jwt"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmail     = errors.New("geçerli bir e-posta adresi giriniz")
	ErrPasswordTooShort = errors.New("şifre en az 8 karakter olmalıdır")
	ErrPasswordMismatch = errors.New("şifreler eşleşmiyor")
	ErrEmailExists      = errors.New("bu e-posta adresi zaten kayıtlı")
	ErrInvalidCredentials = errors.New("e-posta veya şifre hatalı")
	ErrUserNotActive    = errors.New("hesap aktif değil")
	ErrFirstNameRequired = errors.New("ad alanı zorunludur")
	ErrLastNameRequired  = errors.New("soyad alanı zorunludur")
)

// AuthService defines the interface for authentication operations.
type AuthService interface {
	Register(req model.RegisterRequest) (*model.AuthResponse, error)
	Login(req model.LoginRequest) (*model.AuthResponse, error)
	UpdateProfile(userID int, req model.UpdateProfileRequest) (*model.User, error)
}

// authService is the concrete implementation.
type authService struct {
	userRepo repository.UserRepository
}

// NewAuthService creates a new AuthService with its dependencies injected.
func NewAuthService(repo repository.UserRepository) AuthService {
	return &authService{userRepo: repo}
}

// Register creates a new user account and returns a JWT token.
func (s *authService) Register(req model.RegisterRequest) (*model.AuthResponse, error) {
	// --- Validation ---
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !isValidEmail(req.Email) {
		return nil, ErrInvalidEmail
	}
	if len(req.Password) < 8 {
		return nil, ErrPasswordTooShort
	}
	if req.Password != req.PasswordConfirm {
		return nil, ErrPasswordMismatch
	}
	if strings.TrimSpace(req.FirstName) == "" {
		return nil, ErrFirstNameRequired
	}
	if strings.TrimSpace(req.LastName) == "" {
		return nil, ErrLastNameRequired
	}

	// --- Business rules ---
	if s.userRepo.EmailExists(req.Email) {
		return nil, ErrEmailExists
	}

	// Hash password with bcrypt (cost=12 for security)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, errors.New("şifre işlenirken hata oluştu")
	}

	// Create user
	user := &model.User{
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		Phone:     strings.TrimSpace(req.Phone),
		Role:      "customer",
	}

	created, err := s.userRepo.Create(user)
	if err != nil {
		return nil, errors.New("kullanıcı oluşturulamadı")
	}

	// Generate JWT — only AuthService does this
	token, err := sharedJWT.GenerateToken(created.ID, created.Email, created.Role, created.FirstName, created.LastName)
	if err != nil {
		return nil, errors.New("token oluşturulamadı")
	}

	return buildAuthResponse(token, created), nil
}

// Login validates credentials and returns a JWT token.
func (s *authService) Login(req model.LoginRequest) (*model.AuthResponse, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		// Don't reveal if user exists — always same error
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrUserNotActive
	}

	// bcrypt comparison — timing-safe
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate JWT
	token, err := sharedJWT.GenerateToken(user.ID, user.Email, user.Role, user.FirstName, user.LastName)
	if err != nil {
		return nil, errors.New("token oluşturulamadı")
	}

	return buildAuthResponse(token, user), nil
}

// buildAuthResponse constructs the auth response DTO.
func buildAuthResponse(token string, user *model.User) *model.AuthResponse {
	resp := &model.AuthResponse{
		Token:     token,
		ExpiresIn: 86400, // 24 hours in seconds
	}
	resp.User.ID = user.ID
	resp.User.Email = user.Email
	resp.User.FirstName = user.FirstName
	resp.User.LastName = user.LastName
	resp.User.Role = user.Role
	return resp
}

// isValidEmail checks if an email address has valid format.
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return re.MatchString(email)
}

// UpdateProfile updates user info and handles password hashing if provided.
func (s *authService) UpdateProfile(userID int, req model.UpdateProfileRequest) (*model.User, error) {
	data := make(map[string]interface{})

	if strings.TrimSpace(req.FirstName) != "" {
		data["first_name"] = strings.TrimSpace(req.FirstName)
	}
	if strings.TrimSpace(req.LastName) != "" {
		data["last_name"] = strings.TrimSpace(req.LastName)
	}

	if req.Password != "" {
		if len(req.Password) < 8 {
			return nil, ErrPasswordTooShort
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
		if err != nil {
			return nil, errors.New("şifre işlenirken hata oluştu")
		}
		data["password"] = string(hashedPassword)
	}

	updated, err := s.userRepo.Update(userID, data)
	if err != nil {
		return nil, errors.New("profil güncellenemedi")
	}

	// We strip sensitive info before returning
	safeUser := *updated
	safeUser.Password = ""
	return &safeUser, nil
}
