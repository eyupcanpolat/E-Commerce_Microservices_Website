// Package repository handles data access for OrderService.
package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"eticaret/order-service/internal/model"
)

// OrderRepository defines the interface for order data operations.
type OrderRepository interface {
	GetByUserID(userID int) ([]model.Order, error)
	GetByID(id int) (*model.Order, error)
	GetByOrderNumber(orderNumber string) (*model.Order, error)
	Create(order *model.Order) (*model.Order, error)
	UpdateStatus(id int, status model.OrderStatus) (*model.Order, error)
	Cancel(id, userID int) error
}

type jsonOrderRepository struct {
	filePath string
	mu       sync.RWMutex
}

func NewOrderRepository(filePath string) OrderRepository {
	return &jsonOrderRepository{filePath: filePath}
}

func (r *jsonOrderRepository) load() ([]model.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read orders file: %w", err)
	}
	var orders []model.Order
	return orders, json.Unmarshal(data, &orders)
}

func (r *jsonOrderRepository) save(orders []model.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0644)
}

func (r *jsonOrderRepository) GetByUserID(userID int) ([]model.Order, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	var result []model.Order
	for _, o := range all {
		if o.UserID == userID {
			result = append(result, o)
		}
	}
	return result, nil
}

func (r *jsonOrderRepository) GetByID(id int) (*model.Order, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, errors.New("order not found")
}

func (r *jsonOrderRepository) GetByOrderNumber(orderNumber string) (*model.Order, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].OrderNumber == orderNumber {
			return &all[i], nil
		}
	}
	return nil, errors.New("order not found")
}

func (r *jsonOrderRepository) Create(order *model.Order) (*model.Order, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}

	maxID := 0
	for _, o := range all {
		if o.ID > maxID {
			maxID = o.ID
		}
	}
	order.ID = maxID + 1
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	order.Status = model.StatusPending
	order.PaymentStatus = "pending"

	// Generate order number: ORD-YEAR-SEQUENCE
	order.OrderNumber = fmt.Sprintf("ORD-%d-%04d", time.Now().Year(), order.ID)

	// Set item IDs
	for i := range order.Items {
		order.Items[i].ID = i + 1
		order.Items[i].OrderID = order.ID
	}

	all = append(all, *order)
	return order, r.save(all)
}

func (r *jsonOrderRepository) UpdateStatus(id int, status model.OrderStatus) (*model.Order, error) {
	all, err := r.load()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			all[i].Status = status
			all[i].UpdatedAt = time.Now()

			now := time.Now()
			if status == model.StatusShipped {
				all[i].ShippedAt = &now
			}
			if status == model.StatusDelivered {
				all[i].DeliveredAt = &now
			}

			if err := r.save(all); err != nil {
				return nil, err
			}
			return &all[i], nil
		}
	}
	return nil, errors.New("order not found")
}

func (r *jsonOrderRepository) Cancel(id, userID int) error {
	all, err := r.load()
	if err != nil {
		return err
	}
	for i := range all {
		if all[i].ID == id && all[i].UserID == userID {
			// Can only cancel pending or processing orders
			if all[i].Status != model.StatusPending && all[i].Status != model.StatusProcessing {
				return errors.New("bu aşamadaki sipariş iptal edilemez")
			}
			all[i].Status = model.StatusCancelled
			all[i].UpdatedAt = time.Now()
			return r.save(all)
		}
	}
	return errors.New("sipariş bulunamadı veya erişim reddedildi")
}
