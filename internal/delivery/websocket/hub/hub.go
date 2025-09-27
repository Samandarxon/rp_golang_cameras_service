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
	Register   chan *Client     // ✅ PUBLIC - Yangi client ro'yxatdan o'tish
	Unregister chan *Client     // ✅ PUBLIC - Client chiqib ketish
	mu         sync.RWMutex     // Thread-safe uchun mutex
	logger     *zap.Logger      // Logger
}

// NewHub - Yangi hub yaratish
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256), // Buffer bilan
		Register:   make(chan *Client),       // ✅ PUBLIC
		Unregister: make(chan *Client),       // ✅ PUBLIC
		logger:     logger,
	}
}

// Run - Hub ni ishga tushirish
func (h *Hub) Run() {
	h.logger.Info("WebSocket Hub ishga tushdi")

	for {
		select {
		case client := <-h.Register:
			// Yangi client ro'yxatdan o'tkazish
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			h.logger.Info("Yangi client ulandi",
				zap.String("client_id", client.ID),
				zap.Int("total_clients", len(h.clients)),
			)

		case client := <-h.Unregister:
			// Client ni o'chirish
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()

			h.logger.Info("Client uzildi",
				zap.String("client_id", client.ID),
				zap.Int("total_clients", len(h.clients)),
			)

		case message := <-h.broadcast:
			// Barcha clientlarga xabar yuborish
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
					// Xabar muvaffaqiyatli yuborildi
				default:
					// Client o'qimayapti, uni o'chiramiz
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast - Barcha clientlarga xabar yuborish
func (h *Hub) Broadcast(message *Message) {
	h.broadcast <- message
}

// Stop - Hub ni to'xtatish
func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Barcha clientlarni yopish
	for client := range h.clients {
		close(client.Send)
		client.Conn.Close()
	}

	h.clients = make(map[*Client]bool)
	h.logger.Info("WebSocket Hub to'xtatildi")
}

// GetClientCount - Faol clientlar sonini olish
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// RegisterClient - Client ni ro'yxatdan o'tkazish (public metod)
func (h *Hub) RegisterClient(client *Client) {
	h.Register <- client
}

// UnregisterClient - Client ni o'chirish (public metod)
func (h *Hub) UnregisterClient(client *Client) {
	h.Unregister <- client
}
