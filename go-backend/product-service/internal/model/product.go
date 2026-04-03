// Package model defines product and category structs for ProductService.
package model

import "time"

// Product maps to the products table in the original SQL schema.
type Product struct {
	ID               int       `json:"id" bson:"_id"`
	CategoryID       *int      `json:"category_id" bson:"category_id"`
	Name             string    `json:"name" bson:"name"`
	Slug             string    `json:"slug" bson:"slug"`
	Description      string    `json:"description" bson:"description"`
	ShortDescription string    `json:"short_description" bson:"short_description"`
	Price            float64   `json:"price" bson:"price"`
	SalePrice        *float64  `json:"sale_price" bson:"sale_price"`
	SKU              string    `json:"sku" bson:"sku"`
	StockQuantity    int       `json:"stock_quantity" bson:"stock_quantity"`
	StockStatus      string    `json:"stock_status" bson:"stock_status"` // in_stock | out_of_stock | low_stock
	ImageURL         string    `json:"image_url" bson:"image_url"`
	GalleryImages    []string  `json:"gallery_images" bson:"gallery_images"`
	IsFeatured       bool      `json:"is_featured" bson:"is_featured"`
	IsActive         bool      `json:"is_active" bson:"is_active"`
	ViewCount        int       `json:"view_count" bson:"view_count"`
	CreatedAt        time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" bson:"updated_at"`
}

// Category maps to the categories table.
type Category struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	ParentID    *int      `json:"parent_id"`
	ImageURL    string    `json:"image_url"`
	IsActive    bool      `json:"is_active"`
	SortOrder   int       `json:"sort_order"`
}

// ProductFilter holds search and filter parameters from query string.
type ProductFilter struct {
	Search     string  `json:"search"`
	CategoryID *int    `json:"category_id"`
	MinPrice   *float64 `json:"min_price"`
	MaxPrice   *float64 `json:"max_price"`
	InStock    bool    `json:"in_stock"`
	IsFeatured *bool   `json:"is_featured"`
	Sort       string  `json:"sort"` // newest | price_asc | price_desc | popular
}

// PaginatedProducts is the response envelope for list endpoints.
type PaginatedProducts struct {
	Data        []Product `json:"data"`
	Total       int       `json:"total"`
	Page        int       `json:"page"`
	PerPage     int       `json:"per_page"`
	TotalPages  int       `json:"total_pages"`
}

// CreateProductRequest is the request body for POST /products (admin only).
type CreateProductRequest struct {
	CategoryID       *int     `json:"category_id"`
	Name             string   `json:"name"`
	Slug             string   `json:"slug"`
	Description      string   `json:"description"`
	ShortDescription string   `json:"short_description"`
	Price            float64  `json:"price"`
	SalePrice        *float64 `json:"sale_price"`
	SKU              string   `json:"sku"`
	StockQuantity    int      `json:"stock_quantity"`
	ImageURL         string   `json:"image_url"`
	GalleryImages    []string `json:"gallery_images"`
	IsFeatured       bool     `json:"is_featured"`
}
