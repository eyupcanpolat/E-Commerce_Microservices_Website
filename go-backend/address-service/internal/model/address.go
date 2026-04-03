// Package model defines address structs for AddressService.
package model

import "time"

// Address maps to the addresses table in the original SQL schema.
type Address struct {
	ID           int       `json:"id" bson:"_id"`
	UserID       int       `json:"user_id" bson:"user_id"`
	Title        string    `json:"title" bson:"title"`
	FirstName    string    `json:"first_name" bson:"first_name"`
	LastName     string    `json:"last_name" bson:"last_name"`
	Phone        string    `json:"phone" bson:"phone"`
	AddressLine1 string    `json:"address_line1" bson:"address_line1"`
	AddressLine2 string    `json:"address_line2" bson:"address_line2"`
	City         string    `json:"city" bson:"city"`
	State        string    `json:"state" bson:"state"`
	PostalCode   string    `json:"postal_code" bson:"postal_code"`
	Country      string    `json:"country" bson:"country"`
	IsDefault    bool      `json:"is_default" bson:"is_default"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
}

// AddressRequest is the payload for CREATE and UPDATE operations.
type AddressRequest struct {
	Title        string `json:"title"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Phone        string `json:"phone"`
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city"`
	State        string `json:"state"`
	PostalCode   string `json:"postal_code"`
	Country      string `json:"country"`
	IsDefault    bool   `json:"is_default"`
}
