// Package tests contains unit tests for AuthService business logic.
// Tests use table-driven test pattern and mock repositories.
package tests

import (
	"errors"
	"testing"

	"eticaret/auth-service/internal/model"
	"eticaret/auth-service/internal/service"
)

// --- Mock Repository ---
// We implement the UserRepository interface with in-memory data for tests.

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
	return nil, nil
}

// --- Tests ---

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
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Token == "" {
		t.Error("expected non-empty JWT token")
	}
	if result.User.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", result.User.Email)
	}
	if result.User.Role != "customer" {
		t.Errorf("expected role customer, got %s", result.User.Role)
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
	if err == nil {
		t.Fatal("expected error for password mismatch, got nil")
	}
	if err != service.ErrPasswordMismatch {
		t.Errorf("expected ErrPasswordMismatch, got: %v", err)
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
				t.Errorf("email %q: expected ErrInvalidEmail, got %v", tt.email, err)
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
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
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

	// First registration should succeed
	_, err := svc.Register(req)
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	// Second registration with same email should fail
	_, err = svc.Register(req)
	if err != service.ErrEmailExists {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc := service.NewAuthService(newMockRepo())

	req := model.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "anypassword",
	}

	_, err := svc.Login(req)
	if err != service.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
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
		t.Errorf("expected ErrFirstNameRequired, got %v", err)
	}
}
