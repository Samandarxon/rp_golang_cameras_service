package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"sale-service/internal/delivery/websocket/hub"
	"sale-service/internal/entity"
)

type productRepository struct {
	db     *pgxpool.Pool // PostgreSQL connection pool
	logger *zap.Logger   // Logger
	hub    *hub.Hub      // WebSocket hub
}

// NewProductRepository - Yangi Product repository yaratish
func NewProductRepository(db *pgxpool.Pool, hub *hub.Hub, logger *zap.Logger) *productRepository {
	return &productRepository{
		db:     db,
		hub:    hub,
		logger: logger,
	}
}

// notifyClients - WebSocket orqali clientlarga xabar yuborish
func (r *productRepository) notifyClients(action string, product *entity.Product) {
	// 🔴 AVVAL HUB NIL EMASLIGINI TEKSHIRAMIZ
	if r.hub == nil {
		r.logger.Error("❌ HUB IS NIL - WebSocket xabar yuborib bo'lmaydi")
		return
	}

	// Debug log
	r.logger.Info("🔔 WebSocket xabar yuborilmoqda",
		zap.String("action", action),
		zap.String("product_id", product.ID),
		zap.String("product_name", product.Name),
		zap.Any("hub", r.hub), // Hub nil emasligini tekshirish
	)

	message := &hub.Message{
		Type:      "product",
		Action:    action,
		Data:      product,
		Timestamp: time.Now(),
	}

	// Barcha clientlarga xabar yuborish
	r.hub.Broadcast(message)

	r.logger.Debug("WebSocket xabari yuborildi",
		zap.String("action", action),
		zap.String("product_id", product.ID),
		zap.String("product_name", product.Name),
	)

	// Debug log
	r.logger.Info("✅ WebSocket xabari yuborildi",
		zap.String("action", action),
		zap.String("product_id", product.ID),
		zap.String("product_name", product.Name),
	)

}

// Create - Yangi mahsulot yaratish
func (r *productRepository) Create(ctx context.Context, product *entity.Product) error {
	// Debug log
	r.logger.Info("🎯 Yangi mahsulot yaratish boshlandi",
		zap.String("product_id", product.ID),
		zap.String("product_name", product.Name),
	)

	query := `
		INSERT INTO products (
			id, name, description, price, quantity, 
			category_id, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	// SQL so'rovni bajarish
	_, err := r.db.Exec(ctx, query,
		product.ID,
		product.Name,
		product.Description,
		product.Price,
		product.Quantity,
		product.CategoryID,
		product.IsActive,
		product.CreatedAt,
		product.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("❌ Database ga saqlashda xatolik",
			zap.Error(err),
			zap.String("product_id", product.ID),
		)
		return fmt.Errorf("mahsulotni database ga saqlashda xatolik: %w", err)
	}

	r.logger.Info("✅ Mahsulot database ga saqlandi, WebSocket xabar yuborilmoqda...")

	// ✅ WebSocket orqali barcha clientlarga bildirish
	r.notifyClients("created", product)

	r.logger.Info("🎉 Mahsulot yaratish tugadi")

	return nil
}

// GetByID - Mahsulotni ID bo'yicha olish
func (r *productRepository) GetByID(ctx context.Context, id string) (*entity.Product, error) {
	query := `
		SELECT id, name, description, price, quantity, 
		       category_id, is_active, created_at, updated_at
		FROM products
		WHERE id = $1 AND is_active = true
	`

	var product entity.Product

	// SQL so'rovni bajarish va natijani scan qilish
	err := r.db.QueryRow(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Quantity,
		&product.CategoryID,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("mahsulot topilmadi: %s", id)
		}
		r.logger.Error("Database dan o'qishda xatolik",
			zap.Error(err),
			zap.String("product_id", id),
		)
		return nil, fmt.Errorf("database dan o'qishda xatolik: %w", err)
	}

	return &product, nil
}

// Update - Mahsulotni yangilash
func (r *productRepository) Update(ctx context.Context, product *entity.Product) error {
	query := `
		UPDATE products
		SET name = $1, description = $2, price = $3, 
		    quantity = $4, category_id = $5, is_active = $6, 
		    updated_at = $7
		WHERE id = $8
	`

	// SQL so'rovni bajarish
	result, err := r.db.Exec(ctx, query,
		product.Name,
		product.Description,
		product.Price,
		product.Quantity,
		product.CategoryID,
		product.IsActive,
		product.UpdatedAt,
		product.ID,
	)

	if err != nil {
		r.logger.Error("Database ni yangilashda xatolik",
			zap.Error(err),
			zap.String("product_id", product.ID),
		)
		return fmt.Errorf("mahsulotni yangilashda xatolik: %w", err)
	}

	// Hech qanday qator yangilanmagan bo'lsa
	if result.RowsAffected() == 0 {
		return fmt.Errorf("mahsulot topilmadi: %s", product.ID)
	}

	// ✅ WebSocket orqali barcha clientlarga bildirish
	r.notifyClients("updated", product)

	return nil
}

// Delete - Mahsulotni o'chirish (soft delete)
func (r *productRepository) Delete(ctx context.Context, id string) error {
	// Avval mahsulotni o'qib olamiz (xabar uchun)
	product, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Hard delete o'rniga soft delete qilamiz (is_active = false)
	query := `
		UPDATE products
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error("Database dan o'chirishda xatolik",
			zap.Error(err),
			zap.String("product_id", id),
		)
		return fmt.Errorf("mahsulotni o'chirishda xatolik: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("mahsulot topilmadi: %s", id)
	}

	// ✅ WebSocket orqali barcha clientlarga bildirish
	r.notifyClients("deleted", product)

	return nil
}

// List - Mahsulotlar ro'yxatini olish (pagination va filter bilan)
func (r *productRepository) List(ctx context.Context, req *entity.ListProductsRequest) ([]*entity.Product, int64, error) {
	// 1. WHERE shartlarni yaratish
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Faqat faol mahsulotlar
	conditions = append(conditions, "is_active = true")

	// Qidiruv (agar berilgan bo'lsa)
	if req.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+req.Search+"%")
		argIndex++
	}

	// Kategoriya bo'yicha filter
	if req.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIndex))
		args = append(args, req.Category)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// 2. Jami mahsulotlar sonini olish
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products WHERE %s", whereClause)

	var totalCount int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		r.logger.Error("Count queryda xatolik", zap.Error(err))
		return nil, 0, fmt.Errorf("count queryda xatolik: %w", err)
	}

	// 3. Mahsulotlar ro'yxatini olish
	// OFFSET va LIMIT ni qo'shish
	offset := (req.Page - 1) * req.Limit

	selectQuery := fmt.Sprintf(`
		SELECT id, name, description, price, quantity, 
		       category_id, is_active, created_at, updated_at
		FROM products
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, req.SortBy, strings.ToUpper(req.SortOrder), argIndex, argIndex+1)

	args = append(args, req.Limit, offset)

	// SQL so'rovni bajarish
	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		r.logger.Error("Query da xatolik", zap.Error(err))
		return nil, 0, fmt.Errorf("query da xatolik: %w", err)
	}
	defer rows.Close()

	// Natijalarni yig'ish

	products := make([]*entity.Product, 0)
	for rows.Next() {
		var product entity.Product
		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.Quantity,
			&product.CategoryID,
			&product.IsActive,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Scan da xatolik", zap.Error(err))
			continue
		}
		products = append(products, &product)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return products, totalCount, nil
}

// List - Mahsulotlar ro'yxatini olish (pagination va filter bilan)
func (r *productRepository) ListAll(ctx context.Context) ([]*entity.Product, int64, error) {

	selectQuery := `
		SELECT COUNT(*) OVER(), id, name, description, price, quantity, 
		       category_id, is_active, created_at, updated_at
		FROM products
	`

	// SQL so'rovni bajarish
	rows, err := r.db.Query(ctx, selectQuery)
	if err != nil {
		r.logger.Error("Query da xatolik", zap.Error(err))
		return nil, 0, fmt.Errorf("query da xatolik: %w", err)
	}
	defer rows.Close()

	// Natijalarni yig'ish
	var totalCount int64
	products := make([]*entity.Product, 0)
	for rows.Next() {
		var product entity.Product
		err := rows.Scan(
			&totalCount,
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.Quantity,
			&product.CategoryID,
			&product.IsActive,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Scan da xatolik", zap.Error(err))
			continue
		}
		products = append(products, &product)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return products, totalCount, nil
}
