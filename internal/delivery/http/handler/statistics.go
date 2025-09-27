// ====================================================================
// FAYL: internal/delivery/http/handler/statistics.go
// MAQSAD: Real-time statistika handler
// ====================================================================
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sale-service/pkg/utils"
)

type statisticsHandler struct {
	logger *zap.Logger
}

// NewStatisticsHandler - Yangi Statistics handler yaratish
func NewStatisticsHandler(logger *zap.Logger) *statisticsHandler {
	return &statisticsHandler{
		logger: logger,
	}
}

// GetRealTimeStats - Real-time statistikani olish
func (h *statisticsHandler) GetRealTimeStats(c *gin.Context) {
	// Global statistika o'zgaruvchilari (bu yerda sodda misol)
	stats := map[string]interface{}{
		"total_requests":           1000,
		"successful_requests":      980,
		"failed_requests":          20,
		"products_created":         50,
		"products_updated":         30,
		"products_deleted":         10,
		"kafka_messages_received":  100,
		"kafka_messages_processed": 95,
		"websocket_connections":    15,
	}

	c.JSON(http.StatusOK, utils.SuccessResponse("Real-time statistika", stats))
}
