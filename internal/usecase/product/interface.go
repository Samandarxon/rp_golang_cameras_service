package product

import (
	"context"
	"sale-service/internal/entity"
)

// Repository - Product repository interface
// Bu interface bizga test yozishda mock repository ishlatishga imkon beradi
type Repository interface {
	Create(ctx context.Context, product *entity.Product) error
	GetByID(ctx context.Context, id string) (*entity.Product, error)
	Update(ctx context.Context, product *entity.Product) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, req *entity.ListProductsRequest) ([]*entity.Product, int64, error)
}

// UseCase - Product use case interface
type UseCase interface {
	Create(ctx context.Context, req *entity.CreateProductRequest) (*entity.Product, error)
	GetByID(ctx context.Context, id string) (*entity.Product, error)
	Update(ctx context.Context, id string, req *entity.UpdateProductRequest) (*entity.Product, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, req *entity.ListProductsRequest) (*entity.ListProductsResponse, error)
}
