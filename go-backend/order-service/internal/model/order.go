// Package model defines order structs for OrderService.
package model

import "time"

// OrderStatus represents the lifecycle of an order.
type OrderStatus string

const (
	StatusPending    OrderStatus = "pending"
	StatusProcessing OrderStatus = "processing"
	StatusShipped    OrderStatus = "shipped"
	StatusDelivered  OrderStatus = "delivered"
	StatusCancelled  OrderStatus = "cancelled"
	StatusRefunded   OrderStatus = "refunded"
)

// OrderItem represents a single product line in an order.
type OrderItem struct {
	ID          int     `json:"id" bson:"id"`
	OrderID     int     `json:"order_id" bson:"order_id"`
	ProductID   int     `json:"product_id" bson:"product_id"`
	ProductName string  `json:"product_name" bson:"product_name"`
	ProductSKU  string  `json:"product_sku" bson:"product_sku"`
	Quantity    int     `json:"quantity" bson:"quantity"`
	UnitPrice   float64 `json:"unit_price" bson:"unit_price"`
	TotalPrice  float64 `json:"total_price" bson:"total_price"`
}

// Order maps to the orders + order_items tables from the SQL schema.
type Order struct {
	ID                int         `json:"id" bson:"_id"`
	UserID            int         `json:"user_id" bson:"user_id"`
	OrderNumber       string      `json:"order_number" bson:"order_number"`
	Status            OrderStatus `json:"status" bson:"status"`
	Subtotal          float64     `json:"subtotal" bson:"subtotal"`
	ShippingCost      float64     `json:"shipping_cost" bson:"shipping_cost"`
	Tax               float64     `json:"tax" bson:"tax"`
	Total             float64     `json:"total" bson:"total"`
	ShippingAddressID int         `json:"shipping_address_id" bson:"shipping_address_id"`
	ShippingMethod    string      `json:"shipping_method" bson:"shipping_method"`
	PaymentMethod     string      `json:"payment_method" bson:"payment_method"`
	PaymentStatus     string      `json:"payment_status" bson:"payment_status"`
	PaymentReference  string      `json:"payment_reference" bson:"payment_reference"`
	Notes             string      `json:"notes" bson:"notes"`
	ShippedAt         *time.Time  `json:"shipped_at" bson:"shipped_at"`
	DeliveredAt       *time.Time  `json:"delivered_at" bson:"delivered_at"`
	CreatedAt         time.Time   `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at" bson:"updated_at"`
	Items             []OrderItem `json:"items" bson:"items"`
}

// CreateOrderRequest is the payload for POST /orders
type CreateOrderRequest struct {
	ShippingAddressID int         `json:"shipping_address_id"`
	ShippingMethod    string      `json:"shipping_method"`
	PaymentMethod     string      `json:"payment_method"`
	Notes             string      `json:"notes"`
	Items             []OrderItemRequest `json:"items"`
}

// OrderItemRequest is a single item in the order creation request.
type OrderItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

// UpdateStatusRequest is used by admin to change order status.
type UpdateStatusRequest struct {
	Status OrderStatus `json:"status"`
}
