// Package repository handles all data access for AuthService.
// Uses a JSON file as a persistent store (mock database).
package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"eticaret/auth-service/internal/model"
)

// UserRepository defines the interface for user data operations.
// Using an interface here allows swapping to a real DB later.
type UserRepository interface {
	FindByEmail(email string) (*model.User, error)
	FindByID(id int) (*model.User, error)
	Create(user *model.User) (*model.User, error)
	Update(id int, data map[string]interface{}) (*model.User, error)
	EmailExists(email string) bool
}

// jsonUserRepository implements UserRepository using a JSON file.
type jsonUserRepository struct {
	filePath string
	mu       sync.RWMutex // protects concurrent reads/writes
}

// NewUserRepository creates a new JSON-backed user repository.
func NewUserRepository(filePath string) UserRepository {
	return &jsonUserRepository{filePath: filePath}
}

// loadUsers reads and deserializes all users from the JSON file.
func (r *jsonUserRepository) loadUsers() ([]model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read users file: %w", err)
	}

	var users []model.User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("failed to parse users JSON: %w", err)
	}
	return users, nil
}

// saveUsers serializes and writes all users back to the JSON file.
func (r *jsonUserRepository) saveUsers(users []model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize users: %w", err)
	}
	return os.WriteFile(r.filePath, data, 0644)
}

func (r *jsonUserRepository) FindByEmail(email string) (*model.User, error) {
	users, err := r.loadUsers()
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].Email == email {
			return &users[i], nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *jsonUserRepository) FindByID(id int) (*model.User, error) {
	users, err := r.loadUsers()
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].ID == id {
			return &users[i], nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *jsonUserRepository) EmailExists(email string) bool {
	_, err := r.FindByEmail(email)
	return err == nil
}

func (r *jsonUserRepository) Create(user *model.User) (*model.User, error) {
	users, err := r.loadUsers()
	if err != nil {
		return nil, err
	}

	// Auto-increment ID
	maxID := 0
	for _, u := range users {
		if u.ID > maxID {
			maxID = u.ID
		}
	}
	user.ID = maxID + 1
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.Role == "" {
		user.Role = "customer"
	}
	user.IsActive = true

	users = append(users, *user)
	if err := r.saveUsers(users); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *jsonUserRepository) Update(id int, data map[string]interface{}) (*model.User, error) {
	users, err := r.loadUsers()
	if err != nil {
		return nil, err
	}

	for i := range users {
		if users[i].ID == id {
			if v, ok := data["first_name"]; ok {
				users[i].FirstName = v.(string)
			}
			if v, ok := data["last_name"]; ok {
				users[i].LastName = v.(string)
			}
			if v, ok := data["phone"]; ok {
				users[i].Phone = v.(string)
			}
			if v, ok := data["password"]; ok {
				users[i].Password = v.(string)
			}
			users[i].UpdatedAt = time.Now()
			if err := r.saveUsers(users); err != nil {
				return nil, err
			}
			return &users[i], nil
		}
	}
	return nil, errors.New("user not found")
}
