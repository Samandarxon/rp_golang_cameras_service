package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	pb "sale-service/genproto/product"

	"sale-service/config"
	"sale-service/internal/delivery/grpc/handler"
	grpcMiddleware "sale-service/internal/delivery/grpc/middleware"
	httpHandler "sale-service/internal/delivery/http/handler"
	wsHandler "sale-service/internal/delivery/websocket/handler"
	"sale-service/internal/delivery/websocket/hub"
	"sale-service/internal/infrastructure/cache"
	"sale-service/internal/infrastructure/kafka/consumer"
	"sale-service/internal/infrastructure/logger"
	"sale-service/internal/repository/postgres"
	"sale-service/internal/usecase/product"
)

func main() {
	// 1. KONFIGURATSIYANI YUKLASH
	// config.yaml faylidan barcha sozlamalarni o'qiydi
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Konfiguratsiya yuklanmadi: %v", err))
	}

	// 2. LOGGERNI ISHGA TUSHIRISH
	// Loglar ham terminal, ham fayl ga yoziladi
	log, err := logger.New(cfg.Logger)
	if err != nil {
		panic(fmt.Sprintf("Logger ishga tushmadi: %v", err))
	}
	defer log.Sync()

	log.Info("🚀 Mikroservis ishga tushmoqda...",
		zap.String("service", cfg.ServiceName),
		zap.String("version", cfg.Version),
	)

	// 3. POSTGRESQL GA ULANISH
	// Ma'lumotlar bazasi connection pool yaratish
	db, err := postgres.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatal("PostgreSQL ga ulanib bo'lmadi", zap.Error(err))
	}
	defer db.Close()
	log.Info("✅ PostgreSQL ga muvaffaqiyatli ulandi")

	// 4. REDIS CACHE GA ULANISH
	// Tez-tez so'raladigan ma'lumotlarni keshlash uchun
	redisCache, err := cache.NewRedisCache(cfg.Redis)
	if err != nil {
		log.Fatal("Redis ga ulanib bo'lmadi", zap.Error(err))
	}
	defer redisCache.Close()
	log.Info("✅ Redis cache muvaffaqiyatli ulandi")

	// 5. WEBSOCKET HUB NI YARATISH
	// Real-time ma'lumot uzatish uchun WebSocket hub
	wsHub := hub.NewHub(log)
	go wsHub.Run() // Hub ni alohida goroutine da ishga tushirish
	log.Info("✅ WebSocket Hub ishga tushdi")

	// 6. REPOSITORY VA USECASE LARNI YARATISH
	// Repository - DB bilan ishlaydi
	// UseCase - business logika
	productRepo := postgres.NewProductRepository(db, wsHub, log)
	productUC := product.NewProductUseCase(productRepo, redisCache, log)

	// 7. KAFKA CONSUMER NI ISHGA TUSHIRISH
	// Kafka dan kelgan xabarlarni qabul qilish va qayta ishlash
	kafkaConsumer := consumer.NewKafkaConsumer(cfg.Kafka, productUC, wsHub, log)
	go kafkaConsumer.Start() // Consumer ni alohida goroutine da ishga tushirish
	log.Info("✅ Kafka Consumer ishga tushdi")

	// 8. HTTP SERVER (Gin) NI SOZLASH
	// REST API endpoints va health check uchun
	ginRouter := setupGinRouter(cfg, productUC, wsHub, log)

	httpServer := &http.Server{
		Addr:         cfg.HTTP.Port,
		Handler:      ginRouter,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	// HTTP serverni alohida goroutine da ishga tushirish
	go func() {
		log.Info("🌐 HTTP Server ishga tushmoqda", zap.String("port", cfg.HTTP.Port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP Server ishga tushmadi", zap.Error(err))
		}
	}()

	// 9. gRPC SERVER NI SOZLASH
	// Boshqa mikroservislar bilan aloqa uchun
	grpcServer := setupGRPCServer(cfg, productUC, log)

	listener, err := net.Listen("tcp", cfg.GRPC.Port)
	if err != nil {
		log.Fatal("gRPC Listener ochilmadi", zap.Error(err))
	}

	// gRPC serverni alohida goroutine da ishga tushirish
	go func() {
		log.Info("📡 gRPC Server ishga tushmoqda", zap.String("port", cfg.GRPC.Port))
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal("gRPC Server ishga tushmadi", zap.Error(err))
		}
	}()

	// 10. GRACEFUL SHUTDOWN
	// SIGINT (Ctrl+C) yoki SIGTERM signalni kutish
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("🛑 Servis to'xtatilmoqda...")

	// Barcha serverlarni to'xtatish
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("HTTP Server to'xtashda xatolik", zap.Error(err))
	}

	grpcServer.GracefulStop()
	kafkaConsumer.Stop()
	wsHub.Stop()

	log.Info("✅ Servis muvaffaqiyatli to'xtatildi")
}

// setupGinRouter - Gin routerini sozlash
func setupGinRouter(cfg *config.Config, productUC product.UseCase, wsHub *hub.Hub, log *zap.Logger) *gin.Engine {
	// Production rejimda ishlatish uchun
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware lar
	router.Use(gin.Recovery())        // Panic dan tiklash
	router.Use(logger.GinLogger(log)) // Har bir request ni loglash
	router.Use(corsMiddleware())      // CORS sozlamalari

	// Health check endpoint
	health := httpHandler.NewHealthHandler(log)
	router.GET("/health", health.Check)
	router.GET("/ready", health.Ready)

	// API versiyalangan endpoint lar
	v1 := router.Group("/api/v1")
	{
		// Product CRUD endpoints
		productHandler := httpHandler.NewProductHandler(productUC, log)
		products := v1.Group("/products")
		{
			products.GET("/all", productHandler.ListAll)   // Barcha mahsulotlar
			products.GET("", productHandler.List)          // Barcha mahsulotlar
			products.GET("/:id", productHandler.GetByID)   // Bitta mahsulot
			products.POST("", productHandler.Create)       // Yangi mahsulot
			products.PUT("/:id", productHandler.Update)    // Mahsulotni yangilash
			products.DELETE("/:id", productHandler.Delete) // Mahsulotni o'chirish
		}

		// Statistics endpoint
		stats := v1.Group("/statistics")
		{
			statsHandler := httpHandler.NewStatisticsHandler(log)
			stats.GET("", statsHandler.GetRealTimeStats) // Real-time statistika
		}
	}

	// WebSocket endpoint
	wsHandler := wsHandler.NewWebSocketHandler(wsHub, log)
	router.GET("/ws", wsHandler.HandleWebSocket)

	return router
}

// setupGRPCServer - gRPC serverini sozlash
func setupGRPCServer(
	cfg *config.Config,
	productUC product.UseCase,
	log *zap.Logger,
) *grpc.Server {
	// gRPC interceptors (middleware)
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcMiddleware.LoggingInterceptor(log),    // Request/Response loglash
			grpcMiddleware.RecoveryInterceptor(log),   // Panic dan tiklash
			grpcMiddleware.ValidationInterceptor(log), // Request validatsiya
		),
	)

	// Product gRPC handler ni ro'yxatdan o'tkazish
	productHandler := handler.NewProductHandler(productUC, log)
	pb.RegisterProductServiceServer(server, productHandler)

	return server
}

// corsMiddleware - CORS sozlamalarini qo'shish
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
