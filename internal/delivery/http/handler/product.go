package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sale-service/internal/entity"
	"sale-service/internal/usecase/product"
	"sale-service/pkg/utils"
)

type productHandler struct {
	productUC product.UseCase // Product use case
	logger    *zap.Logger     // Logger
}

// NewProductHandler - Yangi Product handler yaratish
func NewProductHandler(productUC product.UseCase, logger *zap.Logger) *productHandler {
	return &productHandler{
		productUC: productUC,
		logger:    logger,
	}
}

// Create - Yangi mahsulot yaratish
// @Summary Yangi mahsulot yaratish
// @Tags Products
// @Accept json
// @Produce json
// @Param product body entity.CreateProductRequest true "Mahsulot ma'lumotlari"
// @Success 201 {object} entity.Product
// @Router /api/v1/products [post]
func (h *productHandler) Create(c *gin.Context) {
	var req entity.CreateProductRequest

	// JSON ni parse qilish va validatsiya
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Request validatsiya xatoligi", zap.Error(err))
		c.JSON(http.StatusBadRequest, utils.ErrorResponse("Noto'g'ri ma'lumot formati", err))
		return
	}

	// Use case orqali mahsulot yaratish
	product, err := h.productUC.Create(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Mahsulot yaratishda xatolik", zap.Error(err))
		c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Mahsulot yaratishda xatolik", err))
		return
	}

	// Muvaffaqiyatli javob
	c.JSON(http.StatusCreated, utils.SuccessResponse("Mahsulot muvaffaqiyatli yaratildi", product))
}

// GetByID - Mahsulotni ID bo'yicha olish
// @Summary Mahsulotni olish
// @Tags Products
// @Accept json
// @Produce json
// @Param id path string true "Mahsulot ID"
// @Success 200 {object} entity.Product
// @Router /api/v1/products/{id} [get]
func (h *productHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	// Use case orqali mahsulotni olish
	product, err := h.productUC.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Mahsulot topilmadi", zap.Error(err), zap.String("id", id))
		c.JSON(http.StatusNotFound, utils.ErrorResponse("Mahsulot topilmadi", err))
		return
	}

	c.JSON(http.StatusOK, utils.SuccessResponse("Mahsulot topildi", product))
}

// Update - Mahsulotni yangilash
// @Summary Mahsulotni yangilash
// @Tags Products
// @Accept json
// @Produce json
// @Param id path string true "Mahsulot ID"
// @Param product body entity.UpdateProductRequest true "Yangilanish ma'lumotlari"
// @Success 200 {object} entity.Product
// @Router /api/v1/products/{id} [put]
func (h *productHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req entity.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Request validatsiya xatoligi", zap.Error(err))
		c.JSON(http.StatusBadRequest, utils.ErrorResponse("Noto'g'ri ma'lumot formati", err))
		return
	}

	// Use case orqali mahsulotni yangilash
	product, err := h.productUC.Update(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.Error("Mahsulotni yangilashda xatolik", zap.Error(err))
		c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Mahsulotni yangilashda xatolik", err))
		return
	}

	c.JSON(http.StatusOK, utils.SuccessResponse("Mahsulot muvaffaqiyatli yangilandi", product))
}

// Delete - Mahsulotni o'chirish
// @Summary Mahsulotni o'chirish
// @Tags Products
// @Accept json
// @Produce json
// @Param id path string true "Mahsulot ID"
// @Success 200 {object} utils.Response
// @Router /api/v1/products/{id} [delete]
func (h *productHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// Use case orqali mahsulotni o'chirish
	if err := h.productUC.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("Mahsulotni o'chirishda xatolik", zap.Error(err))
		c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Mahsulotni o'chirishda xatolik", err))
		return
	}

	c.JSON(http.StatusOK, utils.SuccessResponse("Mahsulot muvaffaqiyatli o'chirildi", nil))
}

// List - Mahsulotlar ro'yxatini olish
// @Summary Mahsulotlar ro'yxati
// @Tags Products
// @Accept json
// @Produce json
// @Param page query int false "Sahifa raqami" default(1)
// @Param limit query int false "Har sahifadagi mahsulotlar" default(10)
// @Param search query string false "Qidiruv so'zi"
// @Param category query string false "Kategoriya ID"
// @Success 200 {object} entity.ListProductsResponse
// @Router /api/v1/products [get]
func (h *productHandler) List(c *gin.Context) {
	var req entity.ListProductsRequest

	// Query parametrlarini bind qilish
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("Query parametrlar xatoligi", zap.Error(err))
		c.JSON(http.StatusBadRequest, utils.ErrorResponse("Noto'g'ri query parametrlar", err))
		return
	}

	// Use case orqali ro'yxatni olish
	response, err := h.productUC.List(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Ro'yxatni olishda xatolik", zap.Error(err))
		c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Ro'yxatni olishda xatolik", err))
		return
	}

	c.JSON(http.StatusOK, utils.SuccessResponse("Mahsulotlar ro'yxati", response))
}
