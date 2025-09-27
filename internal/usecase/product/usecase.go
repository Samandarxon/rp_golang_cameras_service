package product

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"sale-service/internal/entity"
	"sale-service/internal/infrastructure/cache"
	"sale-service/pkg/errors"
)

type productUseCase struct {
	repo   Repository  // Repository interface
	cache  cache.Cache // Redis cache
	logger *zap.Logger // Logger
}

// NewProductUseCase - Yangi Product UseCase yaratish
func NewProductUseCase(repo Repository, cache cache.Cache, logger *zap.Logger) UseCase {
	return &productUseCase{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

// Create - Yangi mahsulot yaratish
func (uc *productUseCase) Create(ctx context.Context, req *entity.CreateProductRequest) (*entity.Product, error) {
	// 1. Yangi Product obyekti yaratish
	product := &entity.Product{
		ID:          uuid.New().String(), // Yangi UUID yaratish
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Quantity:    req.Quantity,
		CategoryID:  req.CategoryID,
		IsActive:    true, // Default faol
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 2. Database ga saqlash
	if err := uc.repo.Create(ctx, product); err != nil {
		uc.logger.Error("Mahsulot yaratishda xatolik",
			zap.Error(err),
			zap.String("product_name", req.Name),
		)
		return nil, errors.Wrap(err, "mahsulot yaratib bo'lmadi")
	}

	// 3. Cache ga saqlash (keyingi so'rovlar uchun)
	cacheKey := fmt.Sprintf("product:%s", product.ID)
	if err := uc.cache.Set(ctx, cacheKey, product, 15*time.Minute); err != nil {
		// Cache xatosi critical emas, faqat log yozamiz
		uc.logger.Warn("Cache ga saqlashda xatolik", zap.Error(err))
	}

	uc.logger.Info("Yangi mahsulot yaratildi",
		zap.String("product_id", product.ID),
		zap.String("product_name", product.Name),
	)

	return product, nil
}

// GetByID - Mahsulotni ID bo'yicha olish
func (uc *productUseCase) GetByID(ctx context.Context, id string) (*entity.Product, error) {
	// 1. Avval cache dan tekshiramiz
	cacheKey := fmt.Sprintf("product:%s", id)

	var product entity.Product
	if err := uc.cache.Get(ctx, cacheKey, &product); err == nil {
		// Cache da topildi
		uc.logger.Debug("Mahsulot cache dan olindi", zap.String("product_id", id))
		return &product, nil
	}

	// 2. Cache da yo'q bo'lsa, database dan olamiz
	productFromDB, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Error("Mahsulot topilmadi",
			zap.Error(err),
			zap.String("product_id", id),
		)
		return nil, errors.Wrap(err, "mahsulot topilmadi")
	}

	// 3. Database dan olganimizni cache ga saqlaymiz
	if err := uc.cache.Set(ctx, cacheKey, productFromDB, 15*time.Minute); err != nil {
		uc.logger.Warn("Cache ga saqlashda xatolik", zap.Error(err))
	}

	return productFromDB, nil
}

// Update - Mahsulotni yangilash
func (uc *productUseCase) Update(ctx context.Context, id string, req *entity.UpdateProductRequest) (*entity.Product, error) {
	// 1. Mahsulot mavjudligini tekshirish
	product, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "mahsulot topilmadi")
	}

	// 2. Faqat berilgan maydonlarni yangilash (partial update)
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.Quantity != nil {
		product.Quantity = *req.Quantity
	}
	if req.CategoryID != nil {
		product.CategoryID = *req.CategoryID
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	product.UpdatedAt = time.Now()

	// 3. Database ga saqlash
	if err := uc.repo.Update(ctx, product); err != nil {
		uc.logger.Error("Mahsulotni yangilashda xatolik",
			zap.Error(err),
			zap.String("product_id", id),
		)
		return nil, errors.Wrap(err, "mahsulotni yangilab bo'lmadi")
	}

	// 4. Cache ni yangilash
	cacheKey := fmt.Sprintf("product:%s", id)
	if err := uc.cache.Set(ctx, cacheKey, product, 15*time.Minute); err != nil {
		uc.logger.Warn("Cache ni yangilashda xatolik", zap.Error(err))
	}

	uc.logger.Info("Mahsulot yangilandi",
		zap.String("product_id", product.ID),
		zap.String("product_name", product.Name),
	)

	return product, nil
}

// Delete - Mahsulotni o'chirish
func (uc *productUseCase) Delete(ctx context.Context, id string) error {
	// 1. Database dan o'chirish
	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.logger.Error("Mahsulotni o'chirishda xatolik",
			zap.Error(err),
			zap.String("product_id", id),
		)
		return errors.Wrap(err, "mahsulotni o'chirib bo'lmadi")
	}

	// 2. Cache dan o'chirish
	cacheKey := fmt.Sprintf("product:%s", id)
	if err := uc.cache.Delete(ctx, cacheKey); err != nil {
		uc.logger.Warn("Cache dan o'chirishda xatolik", zap.Error(err))
	}

	uc.logger.Info("Mahsulot o'chirildi", zap.String("product_id", id))

	return nil
}

// List - Mahsulotlar ro'yxatini olish
func (uc *productUseCase) List(ctx context.Context, req *entity.ListProductsRequest) (*entity.ListProductsResponse, error) {
	// Default qiymatlar
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 10
	}
	if req.SortBy == "" {
		req.SortBy = "created_at"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Database dan ro'yxat olish
	products, totalCount, err := uc.repo.List(ctx, req)
	if err != nil {
		uc.logger.Error("Mahsulotlar ro'yxatini olishda xatolik", zap.Error(err))
		return nil, errors.Wrap(err, "mahsulotlar ro'yxatini olib bo'lmadi")
	}

	// Jami sahifalar sonini hisoblash
	totalPages := int(totalCount) / req.Limit
	if int(totalCount)%req.Limit != 0 {
		totalPages++
	}

	response := &entity.ListProductsResponse{
		Products:   products,
		TotalCount: totalCount,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}

	return response, nil
}
