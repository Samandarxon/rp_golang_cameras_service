package handler

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "sale-service/genproto/product"
	"sale-service/internal/entity"
	"sale-service/internal/usecase/product"
)

type productHandler struct {
	pb.UnimplementedProductServiceServer
	productUC product.UseCase
	logger    *zap.Logger
}

// NewProductHandler - Yangi gRPC Product handler yaratish
func NewProductHandler(productUC product.UseCase, logger *zap.Logger) pb.ProductServiceServer {
	return &productHandler{
		productUC: productUC,
		logger:    logger,
	}
}

// CreateProduct - Yangi mahsulot yaratish (gRPC)
func (h *productHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.Product, error) {
	// Request ni entity ga o'girish
	createReq := &entity.CreateProductRequest{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Quantity:    int(req.Quantity),
		CategoryID:  req.CategoryId,
	}

	// Use case orqali mahsulot yaratish
	product, err := h.productUC.Create(ctx, createReq)
	if err != nil {
		h.logger.Error("gRPC: Mahsulot yaratishda xatolik", zap.Error(err))
		return nil, status.Error(codes.Internal, "Mahsulot yaratib bo'lmadi")
	}

	// Entity ni protobuf ga o'girish
	return &pb.Product{
		Id:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Quantity:    int32(product.Quantity),
		CategoryId:  product.CategoryID,
		IsActive:    product.IsActive,
		CreatedAt:   product.CreatedAt.Unix(),
		UpdatedAt:   product.UpdatedAt.Unix(),
	}, nil
}

// GetProduct - Mahsulotni olish (gRPC)
func (h *productHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	product, err := h.productUC.GetByID(ctx, req.Id)
	if err != nil {
		h.logger.Error("gRPC: Mahsulot topilmadi", zap.Error(err))
		return nil, status.Error(codes.NotFound, "Mahsulot topilmadi")
	}

	return &pb.Product{
		Id:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Quantity:    int32(product.Quantity),
		CategoryId:  product.CategoryID,
		IsActive:    product.IsActive,
		CreatedAt:   product.CreatedAt.Unix(),
		UpdatedAt:   product.UpdatedAt.Unix(),
	}, nil
}

// UpdateProduct - Mahsulotni yangilash (gRPC)
func (h *productHandler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.Product, error) {
	// Update request yaratish
	updateReq := &entity.UpdateProductRequest{}

	if req.Name != nil {
		name := req.Name.Value
		updateReq.Name = &name
	}
	if req.Description != nil {
		desc := req.Description.Value
		updateReq.Description = &desc
	}
	if req.Price != nil {
		price := req.Price.Value
		updateReq.Price = &price
	}
	if req.Quantity != nil {
		qty := int(req.Quantity.Value)
		updateReq.Quantity = &qty
	}

	product, err := h.productUC.Update(ctx, req.Id, updateReq)
	if err != nil {
		h.logger.Error("gRPC: Mahsulotni yangilashda xatolik", zap.Error(err))
		return nil, status.Error(codes.Internal, "Mahsulotni yangilab bo'lmadi")
	}

	return &pb.Product{
		Id:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Quantity:    int32(product.Quantity),
		CategoryId:  product.CategoryID,
		IsActive:    product.IsActive,
		CreatedAt:   product.CreatedAt.Unix(),
		UpdatedAt:   product.UpdatedAt.Unix(),
	}, nil
}

// DeleteProduct - Mahsulotni o'chirish (gRPC)
func (h *productHandler) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	if err := h.productUC.Delete(ctx, req.Id); err != nil {
		h.logger.Error("gRPC: Mahsulotni o'chirishda xatolik", zap.Error(err))
		return nil, status.Error(codes.Internal, "Mahsulotni o'chirib bo'lmadi")
	}

	return &pb.DeleteProductResponse{
		Success: true,
		Message: "Mahsulot muvaffaqiyatli o'chirildi",
	}, nil
}

// ListProducts - Mahsulotlar ro'yxati (gRPC)
func (h *productHandler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	listReq := &entity.ListProductsRequest{
		Page:      int(req.Page),
		Limit:     int(req.Limit),
		Search:    req.Search,
		Category:  req.Category,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	response, err := h.productUC.List(ctx, listReq)
	if err != nil {
		h.logger.Error("gRPC: Ro'yxatni olishda xatolik", zap.Error(err))
		return nil, status.Error(codes.Internal, "Ro'yxatni olib bo'lmadi")
	}

	// Entity larni protobuf ga o'girish
	products := make([]*pb.Product, len(response.Products))
	for i, p := range response.Products {
		products[i] = &pb.Product{
			Id:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
			Quantity:    int32(p.Quantity),
			CategoryId:  p.CategoryID,
			IsActive:    p.IsActive,
			CreatedAt:   p.CreatedAt.Unix(),
			UpdatedAt:   p.UpdatedAt.Unix(),
		}
	}

	return &pb.ListProductsResponse{
		Products:   products,
		TotalCount: response.TotalCount,
		Page:       int32(response.Page),
		Limit:      int32(response.Limit),
		TotalPages: int32(response.TotalPages),
	}, nil
}
