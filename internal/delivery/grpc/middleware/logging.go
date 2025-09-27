package middleware

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// LoggingInterceptor - gRPC requestlarni loglash
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Request boshlanish vaqti
		start := time.Now()

		// Handler ni chaqirish
		resp, err := handler(ctx, req)

		// Request tugash vaqti
		duration := time.Since(start)

		// Log yozish
		if err != nil {
			logger.Error("gRPC Request xatolik bilan tugadi",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
		} else {
			logger.Info("gRPC Request",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
			)
		}

		return resp, err
	}
}
