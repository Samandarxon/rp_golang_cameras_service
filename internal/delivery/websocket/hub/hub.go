package hub

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Message - WebSocket xabar strukturasi
type Message struct {
	Type      string      `json:"type"`      // Xabar turi (product, statistics, error)
	Action    string      `json:"action"`    // Action (created, updated, deleted)
	Data      interface{} `json:"data"`      // Xabar ma'lumotlari
	Timestamp time.Time   `json:"timestamp"` // Xabar vaqti
}

// Client - WebSocket client
type Client struct {
	ID   string          // Client ID
	Conn *websocket.Conn // WebSocket connection
	Send chan *Message   // Xabar yuborish kanali
}

// Hub - WebSocket hub
type Hub struct {
	clients    map[*Client]bool // Barcha clientlar
	broadcast  chan *Message    // Broadcast xabar kanali
	Register   chan *Client     // Yangi client ro'yxatdan o'tish
	Unregister chan *Client     // Client chiqib ketish
	mu         sync.RWMutex     // Thread-safe uchun mutex
	logger     *zap.Logger      // Logger
}

// NewHub - Yangi hub yaratish
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256), // Buffer bilan
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run - Hub ni ishga tushirish
func (h *Hub) Run() {
	h.logger.Info("WebSocket Hub ishga tushdi")

	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient - Client ni ro'yxatdan o'tkazish
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true
	h.logger.Info("Yangi client ulandi",
		zap.String("client_id", client.ID),
		zap.Int("total_clients", len(h.clients)),
	)
}

// unregisterClient - Client ni o'chirish
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		h.safeCloseChannel(client.Send)
		h.logger.Info("Client uzildi",
			zap.String("client_id", client.ID),
			zap.Int("total_clients", len(h.clients)),
		)
	}
}

// broadcastMessage - Barcha clientlarga xabar yuborish
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// ✅ O'chiriladigan clientlarni yig'ish
	var clientsToRemove []*Client

	for client := range h.clients {
		select {
		case client.Send <- message:
			// Xabar muvaffaqiyatli yuborildi
		default:
			// Client o'qimayapti, keyin o'chiramiz
			clientsToRemove = append(clientsToRemove, client)
			h.logger.Warn("Client ga xabar yuborib bo'lmadi, o'chiriladi",
				zap.String("client_id", client.ID),
			)
		}
	}

	// ✅ O'chirish kerak bo'lgan clientlarni alohida o'chiramiz
	if len(clientsToRemove) > 0 {
		go h.removeClients(clientsToRemove)
	}
}

// removeClients - Clientlarni o'chirish
func (h *Hub) removeClients(clients []*Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range clients {
		if _, ok := h.clients[client]; ok {
			delete(h.clients, client)
			h.safeCloseChannel(client.Send)
			h.logger.Info("Client broadcast vaqtida o'chirildi",
				zap.String("client_id", client.ID),
				zap.Int("total_clients", len(h.clients)),
			)
		}
	}
}

// safeCloseChannel - Channel ni xavfsiz yopish
func (h *Hub) safeCloseChannel(ch chan *Message) {
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

// Broadcast - Barcha clientlarga xabar yuborish
func (h *Hub) Broadcast(message *Message) {
	h.logger.Info("🔔 HUB: Broadcast xabar",
		zap.String("type", message.Type),
		zap.String("action", message.Action),
		zap.Int("clients_soni", len(h.clients)),
		zap.Time("timestamp", message.Timestamp),
	)

	// ✅ Xavfsiz broadcast
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("Broadcast da panic",
					zap.Any("panic", r),
				)
			}
		}()
		h.broadcast <- message
	}()
}

// Stop - Hub ni to'xtatish
func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.logger.Info("WebSocket Hub to'xtatilmoqda...",
		zap.Int("clients_soni", len(h.clients)),
	)

	// Barcha clientlarni yopish
	for client := range h.clients {
		h.safeCloseChannel(client.Send)
		h.safeCloseConnection(client.Conn)
	}

	// Channel larni yopish
	close(h.Register)
	close(h.Unregister)
	close(h.broadcast)

	h.clients = make(map[*Client]bool)
	h.logger.Info("WebSocket Hub to'xtatildi")
}

// safeCloseConnection - Connection ni xavfsiz yopish
func (h *Hub) safeCloseConnection(conn *websocket.Conn) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Debug("Connection allaqachon yopilgan",
				zap.Any("recover", r),
			)
		}
	}()

	if conn != nil {
		conn.Close()
	}
}

// GetClientCount - Faol clientlar sonini olish
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// RegisterClient - Client ni ro'yxatdan o'tkazish (public metod)
func (h *Hub) RegisterClient(client *Client) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("RegisterClient da panic",
					zap.Any("panic", r),
				)
			}
		}()
		h.Register <- client
	}()
}

// UnregisterClient - Client ni o'chirish (public metod)
func (h *Hub) UnregisterClient(client *Client) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("UnregisterClient da panic",
					zap.Any("panic", r),
				)
			}
		}()
		h.Unregister <- client
	}()
}

// GetClientIDs - Barcha client ID larini olish
func (h *Hub) GetClientIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ids := make([]string, 0, len(h.clients))
	for client := range h.clients {
		ids = append(ids, client.ID)
	}
	return ids
}

// IsClientConnected - Client ulanganligini tekshirish
func (h *Hub) IsClientConnected(clientID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.ID == clientID {
			return true
		}
	}
	return false
}
