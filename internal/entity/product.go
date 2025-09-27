package entity

import (
	"time"
)

// Product - Mahsulot modeli
type Product struct {
	ID          string    `json:"id" db:"id"`                   // Mahsulot ID
	Name        string    `json:"name" db:"name"`               // Mahsulot nomi
	Description string    `json:"description" db:"description"` // Tavsif
	Price       float64   `json:"price" db:"price"`             // Narx
	Quantity    int       `json:"quantity" db:"quantity"`       // Miqdor
	CategoryID  string    `json:"category_id" db:"category_id"` // Kategoriya ID
	IsActive    bool      `json:"is_active" db:"is_active"`     // Faolmi
	CreatedAt   time.Time `json:"created_at" db:"created_at"`   // Yaratilgan vaqt
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`   // Yangilangan vaqt
}

// KafkaEvent - Kafka xabarlari uchun struktura
type KafkaEvent struct {
	EventType string                `json:"event_type"` // create, update, delete
	Data      *CreateProductRequest `json:"data"`       // Xabar ma'lumotlari
	Timestamp time.Time             `json:"timestamp"`  // Xabar vaqti
	TraceID   string                `json:"trace_id"`   // Trace ID
}

// CreateProductRequest - Yangi mahsulot yaratish so'rovi
type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required"` // Majburiy maydon
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`     // Narx 0 dan katta bo'lishi kerak
	Quantity    int     `json:"quantity" binding:"required,gte=0"` // Miqdor 0 dan kichik bo'lmasligi kerak
	CategoryID  string  `json:"category_id" binding:"required"`
}

// UpdateProductRequest - Mahsulotni yangilash so'rovi
type UpdateProductRequest struct {
	Name        *string  `json:"name,omitempty"` // Optional - agar berilmasa o'zgarmasligi kerak
	Description *string  `json:"description,omitempty"`
	Price       *float64 `json:"price,omitempty" binding:"omitempty,gt=0"`
	Quantity    *int     `json:"quantity,omitempty" binding:"omitempty,gte=0"`
	CategoryID  *string  `json:"category_id,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// ListProductsRequest - Mahsulotlar ro'yxatini olish so'rovi
type ListProductsRequest struct {
	Page      int    `json:"page" form:"page"`             // Sahifa raqami
	Limit     int    `json:"limit" form:"limit"`           // Har bir sahifadagi mahsulotlar soni
	Search    string `json:"search" form:"search"`         // Qidiruv so'zi
	Category  string `json:"category" form:"category"`     // Kategoriya bo'yicha filter
	SortBy    string `json:"sort_by" form:"sort_by"`       // Qaysi maydon bo'yicha tartiblash (name, price, created_at)
	SortOrder string `json:"sort_order" form:"sort_order"` // asc yoki desc
}

// ListProductsResponse - Mahsulotlar ro'yxati javobi
type ListProductsResponse struct {
	Products   []*Product `json:"products"`    // Mahsulotlar ro'yxati
	TotalCount int64      `json:"total_count"` // Jami mahsulotlar soni
	Page       int        `json:"page"`        // Joriy sahifa
	Limit      int        `json:"limit"`       // Har bir sahifadagi mahsulotlar soni
	TotalPages int        `json:"total_pages"` // Jami sahifalar soni
}
