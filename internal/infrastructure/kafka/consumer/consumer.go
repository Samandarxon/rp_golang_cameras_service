package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"sale-service/config"
	"sale-service/internal/delivery/websocket/hub"
	"sale-service/internal/entity"
	"sale-service/internal/usecase/product"
)

// Prometheus metrikalari
var (
	kafkaMessagesReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_received_total",
			Help: "Total number of Kafka messages received",
		},
		[]string{"topic", "status"},
	)

	kafkaMessagesProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_processed_total",
			Help: "Total number of Kafka messages processed successfully",
		},
		[]string{"topic", "event_type"},
	)

	kafkaMessagesFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_failed_total",
			Help: "Total number of Kafka messages failed to process",
		},
		[]string{"topic", "event_type", "error_type"},
	)

	kafkaProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_message_processing_duration_seconds",
			Help:    "Duration of Kafka message processing",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic", "event_type", "status"},
	)
)

func init() {
	prometheus.MustRegister(
		kafkaMessagesReceived,
		kafkaMessagesProcessed,
		kafkaMessagesFailed,
		kafkaProcessingDuration,
	)
}

// KafkaEvent - Kafka xabarlari uchun struktura
type KafkaEvent struct {
	EventType string                       `json:"event_type"` // create, update, delete
	Data      *entity.CreateProductRequest `json:"data"`       // Xabar ma'lumotlari
	Timestamp time.Time                    `json:"timestamp"`  // Xabar vaqti
	TraceID   string                       `json:"trace_id"`   // Trace ID
}

// KafkaConsumer - Kafka consumer
type KafkaConsumer struct {
	reader      *kafka.Reader       // Kafka reader
	productUC   product.UseCase     // Product use case
	wsHub       *hub.Hub            // WebSocket hub
	logger      *zap.Logger         // Logger
	wg          sync.WaitGroup      // WaitGroup
	ctx         context.Context     // Context
	cancel      context.CancelFunc  // Cancel function
	stats       Statistics          // Statistika
	config      KafkaConsumerConfig // Sozlamalar
	messageChan chan kafka.Message  // Xabarlar kanali
}

// KafkaConsumerConfig - Consumer sozlamalari
type KafkaConsumerConfig struct {
	config.KafkaConfig
	MaxRetries  int           `yaml:"max_retries"`  // Maksimal qayta urinishlar
	WorkerCount int           `yaml:"worker_count"` // Worker lar soni
	QueueSize   int           `yaml:"queue_size"`   // Xabarlar navbati hajmi
	Timeout     time.Duration `yaml:"timeout"`      // Operation timeout
	RetryDelay  time.Duration `yaml:"retry_delay"`  // Qayta urinish oralig'i
}

// Statistics - Consumer statistikasi
type Statistics struct {
	MessagesReceived  int64 // Qabul qilingan xabarlar
	MessagesProcessed int64 // Qayta ishlangan xabarlar
	MessagesFailed    int64 // Xatolik bilan yakunlangan xabarlar
	WorkersActive     int32 // Faol worker lar soni
}

// NewKafkaConsumer - Yangi Kafka consumer yaratish
func NewKafkaConsumer(
	cfg config.KafkaConfig,
	productUC product.UseCase,
	wsHub *hub.Hub,
	logger *zap.Logger,
) *KafkaConsumer {
	ctx, cancel := context.WithCancel(context.Background())

	// Default sozlamalar
	consumerConfig := KafkaConsumerConfig{
		KafkaConfig: cfg,
		MaxRetries:  3,
		WorkerCount: 5,
		QueueSize:   100,
		Timeout:     30 * time.Second,
		RetryDelay:  1 * time.Second,
	}

	// Kafka reader yaratish
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
		QueueCapacity:  consumerConfig.QueueSize,
	})

	return &KafkaConsumer{
		reader:      reader,
		productUC:   productUC,
		wsHub:       wsHub,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		config:      consumerConfig,
		messageChan: make(chan kafka.Message, consumerConfig.QueueSize),
	}
}

// Start - Consumer ni ishga tushirish
func (kc *KafkaConsumer) Start() {
	// Kafka mavjudligini tekshirish
	if !kc.isKafkaAvailable() {
		kc.logger.Warn("Kafka mavjud emas, consumer offline rejimda ishlaydi")
		go kc.startMockConsumer()
		return
	}

	// Asl Kafka consumer logikasi
	kc.logger.Info("Kafka Consumer ishga tushmoqda",
		zap.Int("workers", kc.config.WorkerCount),
		zap.String("topic", kc.config.Topic),
		zap.Strings("brokers", kc.config.Brokers),
	)

	kc.wg.Add(1)
	go kc.messageReader()

	for i := 0; i < kc.config.WorkerCount; i++ {
		kc.wg.Add(1)
		atomic.AddInt32(&kc.stats.WorkersActive, 1)
		go kc.worker(i)
	}

	kc.logger.Info("Kafka Consumer muvaffaqiyatli ishga tushdi")
}

// isKafkaAvailable - Kafka mavjudligini tekshirish
func (kc *KafkaConsumer) isKafkaAvailable() bool {
	// Birinchi broker ga ulanishni sinab ko'rish
	conn, err := net.DialTimeout("tcp", kc.config.Brokers[0], 3*time.Second)
	if err != nil {
		kc.logger.Warn("Kafka ga ulanish mumkin emas",
			zap.String("broker", kc.config.Brokers[0]),
			zap.Error(err))
		return false
	}
	conn.Close()
	return true
}

// startMockConsumer - Mock consumer ishga tushirish
func (kc *KafkaConsumer) startMockConsumer() {
	kc.logger.Info("Mock consumer ishga tushdi")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-kc.ctx.Done():
			return
		case <-ticker.C:
			// Mock xabar yaratish
			mockMessage := kafka.Message{
				Topic:     kc.config.Topic,
				Partition: 0,
				Offset:    0,
				Key:       []byte("mock-key"),
				Value:     kc.createMockEvent(),
				Time:      time.Now(),
			}

			select {
			case kc.messageChan <- mockMessage:
				kc.logger.Debug("Mock xabar yuborildi")
			case <-kc.ctx.Done():
				return
			}
		}
	}
}

// createMockEvent - Mock event yaratish
func (kc *KafkaConsumer) createMockEvent() []byte {
	event := KafkaEvent{
		EventType: "create",
		Data: &entity.CreateProductRequest{
			Name:        fmt.Sprintf("Test Product %d", time.Now().Unix()),
			Description: "Mock product for testing",
			Price:       100.0,
			CategoryID:  "test-category-id", // Category o'rniga CategoryID
			Quantity:    10,                 // Stock o'rniga Quantity
		},
		Timestamp: time.Now(),
		TraceID:   fmt.Sprintf("mock_trace_%d", time.Now().UnixNano()),
	}

	data, _ := json.Marshal(event)
	return data
}

// messageReader - Xabarlarni o'qib navbatga joylash
func (kc *KafkaConsumer) messageReader() {
	defer kc.wg.Done()

	for {
		select {
		case <-kc.ctx.Done():
			kc.logger.Info("Message reader to'xtatilmoqda")
			close(kc.messageChan)
			return
		default:
			msg, err := kc.reader.ReadMessage(kc.ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				kc.logger.Error("Kafka dan xabar o'qishda xatolik", zap.Error(err))
				kafkaMessagesReceived.WithLabelValues(kc.config.Topic, "error").Inc()
				continue
			}

			// Xabarni navbatga joylash
			select {
			case kc.messageChan <- msg:
				atomic.AddInt64(&kc.stats.MessagesReceived, 1)
				kafkaMessagesReceived.WithLabelValues(kc.config.Topic, "success").Inc()

				kc.logger.Debug("Xabar navbatga qo'shildi",
					zap.String("topic", msg.Topic),
					zap.Int64("offset", msg.Offset),
				)
			case <-kc.ctx.Done():
				return
			default:
				// Navbat to'ligan
				kc.logger.Warn("Xabarlar navbati to'ligan, xabar tashlab yuborildi")
				kafkaMessagesReceived.WithLabelValues(kc.config.Topic, "dropped").Inc()
			}
		}
	}
}

// worker - Xabarlarni qayta ishlash
func (kc *KafkaConsumer) worker(id int) {
	defer func() {
		atomic.AddInt32(&kc.stats.WorkersActive, -1)
		kc.wg.Done()
	}()

	kc.logger.Debug("Worker ishga tushdi", zap.Int("worker_id", id))

	for msg := range kc.messageChan {
		// Context bekor qilinganmi tekshirish
		if kc.ctx.Err() != nil {
			return
		}

		// Xabarni qayta ishlash
		kc.processMessageWithRetry(msg)
	}

	kc.logger.Debug("Worker to'xtadi", zap.Int("worker_id", id))
}

// processMessageWithRetry - Xabarni qayta urinishlar bilan qayta ishlash
func (kc *KafkaConsumer) processMessageWithRetry(msg kafka.Message) {
	startTime := time.Now()
	traceID := generateTraceID()

	defer func() {
		duration := time.Since(startTime).Seconds()
		kafkaProcessingDuration.WithLabelValues(msg.Topic, "unknown", "completed").Observe(duration)
	}()

	for attempt := 0; attempt <= kc.config.MaxRetries; attempt++ {
		err := kc.processMessage(msg, traceID)
		if err == nil {
			// Muvaffaqiyatli
			atomic.AddInt64(&kc.stats.MessagesProcessed, 1)
			return
		}

		if attempt == kc.config.MaxRetries {
			// Maksimal urinishlardan keyin ham xatolik
			kc.logger.Error("Xabarni qayta ishlashda xatolik (maksimal urinishlar)",
				zap.String("trace_id", traceID),
				zap.String("topic", msg.Topic),
				zap.Int64("offset", msg.Offset),
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)

			atomic.AddInt64(&kc.stats.MessagesFailed, 1)
			kafkaMessagesFailed.WithLabelValues(msg.Topic, "unknown", "max_retries_exceeded").Inc()

			// Failed message ni log qilish yoki DLQ ga yuborish
			kc.handleFailedMessage(msg, err, traceID)
			return
		}

		kc.logger.Warn("Xabarni qayta ishlashda xatolik, qayta urinilmoqda",
			zap.String("trace_id", traceID),
			zap.String("topic", msg.Topic),
			zap.Int64("offset", msg.Offset),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
			zap.Duration("retry_delay", kc.config.RetryDelay*time.Duration(attempt+1)),
		)

		// Exponential backoff
		time.Sleep(kc.config.RetryDelay * time.Duration(attempt+1))
	}
}

// processMessage - Xabarni qayta ishlash
func (kc *KafkaConsumer) processMessage(msg kafka.Message, traceID string) error {
	startTime := time.Now()

	kc.logger.Debug("Xabar qayta ishlanmoqda",
		zap.String("trace_id", traceID),
		zap.String("topic", msg.Topic),
		zap.Int("partition", msg.Partition),
		zap.Int64("offset", msg.Offset),
	)

	// JSON ni parse qilish
	var event KafkaEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		kafkaMessagesFailed.WithLabelValues(msg.Topic, "unknown", "parse_error").Inc()
		return fmt.Errorf("xabarni parse qilishda xatolik: %w", err)
	}

	// Event ni tekshirish
	if err := kc.validateEvent(&event); err != nil {
		kafkaMessagesFailed.WithLabelValues(msg.Topic, event.EventType, "validation_error").Inc()
		return fmt.Errorf("event validatsiyada xatolik: %w", err)
	}

	// Trace ID ni yangilash
	if event.TraceID == "" {
		event.TraceID = traceID
	}

	// Context yaratish timeout bilan
	ctx, cancel := context.WithTimeout(context.Background(), kc.config.Timeout)
	defer cancel()

	// Event type bo'yicha qayta ishlash
	var processingErr error
	switch event.EventType {
	case "create":
		processingErr = kc.handleCreateEvent(ctx, &event, traceID)
	case "update":
		processingErr = kc.handleUpdateEvent(ctx, &event, traceID)
	case "delete":
		processingErr = kc.handleDeleteEvent(ctx, &event, traceID)
	default:
		kc.logger.Warn("Noma'lum event type",
			zap.String("trace_id", traceID),
			zap.String("event_type", event.EventType),
		)
		kafkaMessagesFailed.WithLabelValues(msg.Topic, event.EventType, "unknown_event_type").Inc()
		return errors.New("noma'lum event type")
	}

	// Processing duration ni hisoblash
	duration := time.Since(startTime).Seconds()
	status := "success"
	if processingErr != nil {
		status = "error"
		kafkaMessagesFailed.WithLabelValues(msg.Topic, event.EventType, "processing_error").Inc()
	} else {
		kafkaMessagesProcessed.WithLabelValues(msg.Topic, event.EventType).Inc()
	}

	kafkaProcessingDuration.WithLabelValues(msg.Topic, event.EventType, status).Observe(duration)

	return processingErr
}

// validateEvent - Event ni tekshirish
func (kc *KafkaConsumer) validateEvent(event *KafkaEvent) error {
	if event.EventType == "" {
		return errors.New("event_type bo'sh bo'lishi mumkin emas")
	}
	if event.Data == nil {
		return errors.New("data bo'sh bo'lishi mumkin emas")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	return nil
}

// handleCreateEvent - Create event ni qayta ishlash
func (kc *KafkaConsumer) handleCreateEvent(ctx context.Context, event *KafkaEvent, traceID string) error {
	// Yangi mahsulot yaratish
	product, err := kc.productUC.Create(ctx, event.Data)
	if err != nil {
		kc.logger.Error("Mahsulot yaratishda xatolik",
			zap.String("trace_id", traceID),
			zap.Error(err),
		)

		// WebSocket orqali xatolik xabarini yuborish
		kc.wsHub.Broadcast(&hub.Message{
			Type:      "error",
			Action:    "create_failed",
			Data:      map[string]interface{}{"error": err.Error(), "trace_id": traceID},
			Timestamp: time.Now(),
		})
		return err
	}

	// WebSocket orqali yangi mahsulot haqida xabar yuborish
	kc.wsHub.Broadcast(&hub.Message{
		Type:      "product",
		Action:    "created",
		Data:      product,
		Timestamp: time.Now(),
	})

	kc.logger.Info("Mahsulot muvaffaqiyatli yaratildi",
		zap.String("trace_id", traceID),
		zap.String("product_id", product.ID),
		zap.String("product_name", product.Name),
	)

	return nil
}

// handleUpdateEvent - Update event ni qayta ishlash
func (kc *KafkaConsumer) handleUpdateEvent(ctx context.Context, event *KafkaEvent, traceID string) error {
	kc.logger.Info("Update event qabul qilindi",
		zap.String("trace_id", traceID),
	)
	// Update logikasi bu yerda amalga oshiriladi
	return nil
}

// handleDeleteEvent - Delete event ni qayta ishlash
func (kc *KafkaConsumer) handleDeleteEvent(ctx context.Context, event *KafkaEvent, traceID string) error {
	kc.logger.Info("Delete event qabul qilindi",
		zap.String("trace_id", traceID),
	)
	// Delete logikasi bu yerda amalga oshiriladi
	return nil
}

// handleFailedMessage - Xatolik bilan yakunlangan xabarni qayta ishlash
func (kc *KafkaConsumer) handleFailedMessage(msg kafka.Message, err error, traceID string) {
	kc.logger.Error("Xatolik bilan yakunlangan xabar",
		zap.String("trace_id", traceID),
		zap.String("topic", msg.Topic),
		zap.Int64("offset", msg.Offset),
		zap.Error(err),
	)

	// WebSocket orqali xatolik haqida xabar yuborish
	kc.wsHub.Broadcast(&hub.Message{
		Type:      "error",
		Action:    "message_processing_failed",
		Data:      map[string]interface{}{"error": err.Error(), "trace_id": traceID},
		Timestamp: time.Now(),
	})

	// Bu yerda failed message ni DLQ ga yuborish yoki log qilish mumkin
}

// HealthCheck - Consumer holatini tekshirish
func (kc *KafkaConsumer) HealthCheck() error {
	if kc.reader == nil {
		return errors.New("Kafka reader not initialized")
	}

	if kc.ctx.Err() != nil {
		return errors.New("Consumer context cancelled")
	}

	// Worker lar faolmi tekshirish
	if atomic.LoadInt32(&kc.stats.WorkersActive) == 0 {
		return errors.New("No active workers")
	}

	return nil
}

// Stop - Consumer ni to'xtatish
func (kc *KafkaConsumer) Stop() {
	kc.logger.Info("Kafka Consumer to'xtatilmoqda...")

	kc.cancel()       // Context ni bekor qilish
	kc.wg.Wait()      // Barcha goroutine lar tugashini kutish
	kc.reader.Close() // Reader ni yopish

	kc.logger.Info("Kafka Consumer to'xtatildi",
		zap.Int64("received", atomic.LoadInt64(&kc.stats.MessagesReceived)),
		zap.Int64("processed", atomic.LoadInt64(&kc.stats.MessagesProcessed)),
		zap.Int64("failed", atomic.LoadInt64(&kc.stats.MessagesFailed)),
	)
}

// GetStatistics - Statistikani olish
func (kc *KafkaConsumer) GetStatistics() Statistics {
	return Statistics{
		MessagesReceived:  atomic.LoadInt64(&kc.stats.MessagesReceived),
		MessagesProcessed: atomic.LoadInt64(&kc.stats.MessagesProcessed),
		MessagesFailed:    atomic.LoadInt64(&kc.stats.MessagesFailed),
		WorkersActive:     atomic.LoadInt32(&kc.stats.WorkersActive),
	}
}

// generateTraceID - Trace ID yaratish
func generateTraceID() string {
	return fmt.Sprintf("trace_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// WithCustomConfig - Custom sozlamalar bilan consumer yaratish
func (kc *KafkaConsumer) WithCustomConfig(customConfig KafkaConsumerConfig) *KafkaConsumer {
	kc.config = customConfig
	return kc
}
