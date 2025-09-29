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

	h.logger.Info("Yangi WebSocket client ulandi",
		zap.String("client_id", client.ID),
		zap.String("remote_addr", c.Request.RemoteAddr),
		zap.String("user_agent", c.Request.UserAgent()),
	)

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
		// ✅ Recover bilan panic ni ushlash
		if r := recover(); r != nil {
			h.logger.Warn("readPump da panic, lekin recover qilindi",
				zap.String("client_id", client.ID),
				zap.Any("panic", r),
			)
		}

		// Graceful cleanup
		h.hub.Unregister <- client

		// ✅ Channel ni xavfsiz yopish
		h.safeCloseChannel(client.Send)

		// ✅ Connection ni xavfsiz yopish
		h.safeCloseConnection(client.Conn)

		h.logger.Info("Client ulanishi yopildi",
			zap.String("client_id", client.ID),
		)
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
			// WebSocket yopilish xatolarini boshqarish
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNoStatusReceived,
				websocket.CloseNormalClosure,
			) {
				h.logger.Warn("WebSocket noexpected yopildi",
					zap.String("client_id", client.ID),
					zap.Error(err),
				)
			} else {
				h.logger.Debug("WebSocket normal yopildi",
					zap.String("client_id", client.ID),
					zap.Error(err),
				)
			}
			break
		}

		h.logger.Debug("Client dan xabar keldi",
			zap.String("client_id", client.ID),
			zap.String("message", string(message)),
			zap.ByteString("raw_message", message),
		)
	}
}

// writePump - Client ga xabarlarni yozish
func (h *webSocketHandler) writePump(client *hub.Client) {
	// Ping timer
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		// ✅ Recover bilan panic ni ushlash
		if r := recover(); r != nil {
			h.logger.Warn("writePump da panic, lekin recover qilindi",
				zap.String("client_id", client.ID),
				zap.Any("panic", r),
			)
		}

		ticker.Stop()

		// ✅ Connection ni xavfsiz yopish
		h.safeCloseConnection(client.Conn)

		h.logger.Debug("writePump to'xtadi",
			zap.String("client_id", client.ID),
		)
	}()

	for {
		select {
		case message, ok := <-client.Send:
			// Xabar keldi
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub client ni yopdi
				h.logger.Debug("Client channel yopildi",
					zap.String("client_id", client.ID),
				)
				// ✅ Xavfsiz close message yuborish
				h.safeWriteMessage(client.Conn, websocket.CloseMessage, []byte{})
				return
			}

			// Xabarni JSON formatda yuborish
			if err := client.Conn.WriteJSON(message); err != nil {
				h.logger.Error("WebSocket yozishda xatolik",
					zap.String("client_id", client.ID),
					zap.Error(err),
				)
				return
			}

			h.logger.Debug("Client ga xabar yuborildi",
				zap.String("client_id", client.ID),
				zap.Any("message", message),
			)

		case <-ticker.C:
			// Ping xabarini yuborish
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.logger.Debug("Ping yuborishda xatolik (ehtimol client yopildi)",
					zap.String("client_id", client.ID),
					zap.Error(err),
				)
				return
			}
		}
	}
}

// ✅ YANGI METOD: Channel ni xavfsiz yopish
func (h *webSocketHandler) safeCloseChannel(ch chan *hub.Message) {
	defer func() {
		if r := recover(); r != nil {
			// Channel allaqachon yopilgan, ignore qilamiz
			h.logger.Debug("Channel allaqachon yopilgan",
				zap.Any("recover", r),
			)
		}
	}()

	// Faqat ochiq bo'lsa yopamiz
	select {
	case <-ch:
		// Channel allaqachon yopilgan
	default:
		close(ch)
	}
}

// ✅ YANGI METOD: Connection ni xavfsiz yopish
func (h *webSocketHandler) safeCloseConnection(conn *websocket.Conn) {
	defer func() {
		if r := recover(); r != nil {
			// Connection allaqachon yopilgan, ignore qilamiz
			h.logger.Debug("Connection allaqachon yopilgan",
				zap.Any("recover", r),
			)
		}
	}()

	if conn != nil {
		conn.Close()
	}
}

// ✅ YANGI METOD: Xavfsiz xabar yozish
func (h *webSocketHandler) safeWriteMessage(conn *websocket.Conn, messageType int, data []byte) {
	defer func() {
		if r := recover(); r != nil {
			// Connection yopilgan, ignore qilamiz
			h.logger.Debug("Connection yopilgan, xabar yozib bo'lmadi",
				zap.Any("recover", r),
			)
		}
	}()

	if conn != nil {
		conn.WriteMessage(messageType, data)
	}
}
