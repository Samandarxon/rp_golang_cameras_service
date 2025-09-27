package entity

import "time"

// Statistics - Real-time statistika
type Statistics struct {
	TotalRequests          int64     `json:"total_requests"`           // Jami so'rovlar
	SuccessfulRequests     int64     `json:"successful_requests"`      // Muvaffaqiyatli so'rovlar
	FailedRequests         int64     `json:"failed_requests"`          // Xatolik bilan yakunlangan so'rovlar
	ProductsCreated        int64     `json:"products_created"`         // Yaratilgan mahsulotlar
	ProductsUpdated        int64     `json:"products_updated"`         // Yangilangan mahsulotlar
	ProductsDeleted        int64     `json:"products_deleted"`         // O'chirilgan mahsulotlar
	KafkaMessagesReceived  int64     `json:"kafka_messages_received"`  // Kafka dan kelgan xabarlar
	KafkaMessagesProcessed int64     `json:"kafka_messages_processed"` // Qayta ishlangan Kafka xabarlari
	WebSocketConnections   int       `json:"websocket_connections"`    // Faol WebSocket connection lar
	LastUpdated            time.Time `json:"last_updated"`             // Oxirgi yangilanish vaqti
}
