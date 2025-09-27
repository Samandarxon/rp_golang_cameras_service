package middleware

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ValidationInterceptor - Request validatsiya
func ValidationInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Bu yerda request validatsiya logikasi bo'lishi mumkin
		// Masalan: req ni validate qilish

		// Hozircha oddiy handler ni chaqiramiz
		return handler(ctx, req)
	}
}
