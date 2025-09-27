package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"sale-service/internal/delivery/websocket/hub"
)

// Upgrader - HTTP ni WebSocket ga o'zgartirish
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CORS uchun
	CheckOrigin: func(r *http.Request) bool {
		return true // Production da bu sozlamani qattiqroq qilish kerak
	},
}

type webSocketHandler struct {
	hub    *hub.Hub    // WebSocket hub
	logger *zap.Logger // Logger
}

// NewWebSocketHandler - Yangi WebSocket handler yaratish
func NewWebSocketHandler(hub *hub.Hub, logger *zap.Logger) *webSocketHandler {
	return &webSocketHandler{
		hub:    hub,
		logger: logger,
	}
}

// HandleWebSocket - WebSocket connection ni boshqarish
func (h *webSocketHandler) HandleWebSocket(c *gin.Context) {
	// HTTP ni WebSocket ga upgrade qilish
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade xatoligi", zap.Error(err))
		return
	}

	// Yangi client yaratish
	client := &hub.Client{
		ID:   uuid.New().String(),
		Conn: conn,
		Send: make(chan *hub.Message, 256), // Buffer bilan
	}

	// Client ni hub ga ro'yxatdan o'tkazish
	h.hub.Register <- client

	// Read va Write goroutine larni ishga tushirish
	go h.writePump(client)
	go h.readPump(client)
}

const (
	// WebSocket timeout sozlamalari
	writeWait      = 10 * time.Second    // Xabar yozish uchun timeout
	pongWait       = 60 * time.Second    // Pong kutish uchun timeout
	pingPeriod     = (pongWait * 9) / 10 // Ping yuborish oralig'i
	maxMessageSize = 512                 // Maksimal xabar hajmi
)

// readPump - Client dan xabarlarni o'qish
func (h *webSocketHandler) readPump(client *hub.Client) {
	defer func() {
		h.hub.Unregister <- client
		client.Conn.Close()
	}()

	// Sozlamalar
	client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetReadLimit(maxMessageSize)
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Xabarlarni o'qish
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket o'qishda xatolik", zap.Error(err))
			}
			break
		}

		h.logger.Debug("Client dan xabar keldi",
			zap.String("client_id", client.ID),
			zap.String("message", string(message)),
		)
	}
}

// writePump - Client ga xabarlarni yozish
func (h *webSocketHandler) writePump(client *hub.Client) {
	// Ping timer
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			// Xabar keldi
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub client ni yopdi
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Xabarni JSON formatda yuborish
			if err := client.Conn.WriteJSON(message); err != nil {
				h.logger.Error("WebSocket yozishda xatolik", zap.Error(err))
				return
			}

		case <-ticker.C:
			// Ping xabarini yuborish
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
