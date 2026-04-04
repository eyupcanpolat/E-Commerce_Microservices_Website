// Package tests contains unit tests for AuthService business logic.
// Tests use table-driven test pattern and mock repositories.
package tests

import (
	"errors"
	"testing"

	"eticaret/auth-service/internal/model"
	"eticaret/auth-service/internal/service"

	"golang.org/x/crypto/bcrypt"
)

// ── Mock Repository ───────────────────────────────────────────────────────────

type mockUserRepository struct {
	users  []model.User
	nextID int
}

func newMockRepo() *mockUserRepository {
	return &mockUserRepository{nextID: 1}
}

func (m *mockUserRepository) FindByEmail(email string) (*model.User, error) {
	for i := range m.users {
		if m.users[i].Email == email {
			return &m.users[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepository) FindByID(id int) (*model.User, error) {
	for i := range m.users {
		if m.users[i].ID == id {
			return &m.users[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepository) EmailExists(email string) bool {
	_, err := m.FindByEmail(email)
	return err == nil
}

func (m *mockUserRepository) Create(user *model.User) (*model.User, error) {
	user.ID = m.nextID
	m.nextID++
	user.IsActive = true
	m.users = append(m.users, *user)
	return user, nil
}

func (m *mockUserRepository) Update(id int, data map[string]interface{}) (*model.User, error) {
	for i := range m.users {
		if m.users[i].ID == id {
			if v, ok := data["first_name"]; ok {
				m.users[i].FirstName = v.(string)
			}
			if v, ok := data["last_name"]; ok {
				m.users[i].LastName = v.(string)
			}
			if v, ok := data["password"]; ok {
				m.users[i].Password = v.(string)
			}
			return &m.users[i], nil
		}
	}
	return nil, errors.New("not found")
}

// ── Register testleri ─────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	svc := service.NewAuthService(newMockRepo())

	req := model.RegisterRequest{
		Email:           "test@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
		FirstName:       "Test",
		LastName:        "User",
	}

	result, err := svc.Register(req)
	if err != nil {
		t.Fatalf("beklenen hata yok, alınan: %v", err)
	}
	if result.Token == "" {
		t.Error("JWT token boş olmamalı")
	}
	if result.User.Email != "test@example.com" {
		t.Errorf("beklenen email test@example.com, alınan %s", result.User.Email)
	}
	if result.User.Role != "customer" {
		t.Errorf("beklenen rol customer, alınan %s", result.User.Role)
	}
}

func TestRegister_PasswordMismatch(t *testing.T) {
	svc := service.NewAuthService(newMockRepo())

	req := model.RegisterRequest{
		Email:           "test@example.com",
		Password:        "password123",
		PasswordConfirm: "different",
		FirstName:       "Test",
		LastName:        "User",
	}

	_, err := svc.Register(req)
	if err != service.ErrPasswordMismatch {
		t.Errorf("beklenen ErrPasswordMismatch, alınan: %v", err)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	svc := service.NewAuthService(newMockRepo())

	tests := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"no-domain", "user@"},
		{"no-at", "userexample.com"},
		{"spaces", "us er@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := model.RegisterRequest{
				Email:           tt.email,
				Password:        "password123",
				PasswordConfirm: "password123",
				FirstName:       "Test",
				LastName:        "User",
			}
			_, err := svc.Register(req)
			if err != service.ErrInvalidEmail {
				t.Errorf("email %q: beklenen ErrInvalidEmail, alınan %v", tt.email, err)
			}
		})
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	svc := service.NewAuthService(newMockRepo())

	req := model.RegisterRequest{
		Email:           "test@example.com",
		Password:        "abc",
		PasswordConfirm: "abc",
		FirstName:       "Test",
		LastName:        "User",
	}

	_, err := svc.Register(req)
	if err != service.ErrPasswordTooShort {
		t.Errorf("beklenen ErrPasswordTooShort, alınan %v", err)
	}
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewAuthService(repo)

	req := model.RegisterRequest{
		Email:           "existing@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
		FirstName:       "Test",
		LastName:        "User",
	}

	_, err := svc.Register(req)
	if err != nil {
		t.Fatalf("ilk kayıt başarısız: %v", err)
	}

	_, err = svc.Register(req)
	if err != service.ErrEmailExists {
		t.Errorf("beklenen ErrEmailExists, alınan %v", err)
	}
}

func TestRegister_EmptyFirstName(t *testing.T) {
	svc := service.NewAuthService(newMockRepo())

	req := model.RegisterRequest{
		Email:           "test@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
		FirstName:       "",
		LastName:        "User",
	}

	_, err := svc.Register(req)
	if err != service.ErrFirstNameRequired {
		t.Errorf("beklenen ErrFirstNameRequired, alınan %v", err)
	}
}

func TestRegister_EmptyLastName(t *testing.T) {
	svc := service.NewAuthService(newMockRepo())

	req := model.RegisterRequest{
		Email:           "test@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
		FirstName:       "Test",
		LastName:        "",
	}

	_, err := svc.Register(req)
	if err != service.ErrLastNameRequired {
		t.Errorf("beklenen ErrLastNameRequired, alınan %v", err)
	}
}

// ── Login testleri ────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewAuthService(repo)

	// Önce kayıt ol
	_, err := svc.Register(model.RegisterRequest{
		Email:           "login@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
		FirstName:       "Login",
		LastName:        "User",
	})
	if err != nil {
		t.Fatalf("kayıt başarısız: %v", err)
	}

	result, err := svc.Login(model.LoginRequest{
		Email:    "login@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("giriş başarısız: %v", err)
	}
	if result.Token == "" {
		t.Error("JWT token boş olmamalı")
	}
	if result.User.Email != "login@example.com" {
		t.Errorf("beklenen email login@example.com, alınan %s", result.User.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewAuthService(repo)

	_, err := svc.Register(model.RegisterRequest{
		Email:           "user@example.com",
		Password:        "correctpassword",
		PasswordConfirm: "correctpassword",
		FirstName:       "Test",
		LastName:        "User",
	})
	if err != nil {
		t.Fatalf("kayıt başarısız: %v", err)
	}

	_, err = svc.Login(model.LoginRequest{
		Email:    "user@example.com",
		Password: "wrongpassword",
	})
	if err != service.ErrInvalidCredentials {
		t.Errorf("beklenen ErrInvalidCredentials, alınan %v", err)
	}
}

func TestLogin_NonExistentUser(t *testing.T) {
	svc := service.NewAuthService(newMockRepo())

	_, err := svc.Login(model.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "anypassword",
	})
	if err != service.ErrInvalidCredentials {
		t.Errorf("beklenen ErrInvalidCredentials, alınan %v", err)
	}
}

func TestLogin_InactiveAccount(t *testing.T) {
	repo := newMockRepo()

	// Pasif kullanıcı doğrudan repo'ya ekle
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	repo.users = append(repo.users, model.User{
		ID:       1,
		Email:    "inactive@example.com",
		Password: string(hash),
		IsActive: false,
	})

	svc := service.NewAuthService(repo)
	_, err := svc.Login(model.LoginRequest{
		Email:    "inactive@example.com",
		Password: "password123",
	})
	if err != service.ErrUserNotActive {
		t.Errorf("beklenen ErrUserNotActive, alınan %v", err)
	}
}

// ── UpdateProfile testleri ────────────────────────────────────────────────────

func TestUpdateProfile_UpdateName(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewAuthService(repo)

	// Önce kullanıcı oluştur
	result, err := svc.Register(model.RegisterRequest{
		Email:           "profile@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
		FirstName:       "Eski",
		LastName:        "İsim",
	})
	if err != nil {
		t.Fatalf("kayıt başarısız: %v", err)
	}

	updated, err := svc.UpdateProfile(result.User.ID, model.UpdateProfileRequest{
		FirstName: "Yeni",
		LastName:  "İsim",
	})
	if err != nil {
		t.Fatalf("profil güncellenemedi: %v", err)
	}
	if updated.FirstName != "Yeni" {
		t.Errorf("beklenen FirstName 'Yeni', alınan '%s'", updated.FirstName)
	}
	if updated.Password != "" {
		t.Error("şifre response'da görünmemeli")
	}
}

func TestUpdateProfile_PasswordTooShort(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewAuthService(repo)

	result, err := svc.Register(model.RegisterRequest{
		Email:           "profile2@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
		FirstName:       "Test",
		LastName:        "User",
	})
	if err != nil {
		t.Fatalf("kayıt başarısız: %v", err)
	}

	_, err = svc.UpdateProfile(result.User.ID, model.UpdateProfileRequest{
		Password: "abc", // çok kısa
	})
	if err != service.ErrPasswordTooShort {
		t.Errorf("beklenen ErrPasswordTooShort, alınan %v", err)
	}
}

func TestUpdateProfile_PasswordChange(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewAuthService(repo)

	result, err := svc.Register(model.RegisterRequest{
		Email:           "profile3@example.com",
		Password:        "oldpassword",
		PasswordConfirm: "oldpassword",
		FirstName:       "Test",
		LastName:        "User",
	})
	if err != nil {
		t.Fatalf("kayıt başarısız: %v", err)
	}

	_, err = svc.UpdateProfile(result.User.ID, model.UpdateProfileRequest{
		Password: "newpassword123",
	})
	if err != nil {
		t.Fatalf("şifre güncellenemedi: %v", err)
	}

	// Yeni şifreyle giriş yapılabilmeli
	loginResult, err := svc.Login(model.LoginRequest{
		Email:    "profile3@example.com",
		Password: "newpassword123",
	})
	if err != nil {
		t.Fatalf("yeni şifreyle giriş başarısız: %v", err)
	}
	if loginResult.Token == "" {
		t.Error("token boş olmamalı")
	}
}
