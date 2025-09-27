package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sale-service/pkg/utils"
)

type healthHandler struct {
	logger *zap.Logger
}

// NewHealthHandler - Yangi Health handler yaratish
func NewHealthHandler(logger *zap.Logger) *healthHandler {
	return &healthHandler{
		logger: logger,
	}
}

// Check - Servisning ishlayotganligini tekshirish
func (h *healthHandler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, utils.SuccessResponse("Servis ishlayapti", map[string]string{
		"status": "healthy",
	}))
}

// Ready - Servisning tayyor bo'lganligini tekshirish
func (h *healthHandler) Ready(c *gin.Context) {
	// Bu yerda database, redis va boshqa servicelar
	// ishlayotganligini tekshirish mumkin
	c.JSON(http.StatusOK, utils.SuccessResponse("Servis tayyor", map[string]string{
		"status": "ready",
	}))
}
