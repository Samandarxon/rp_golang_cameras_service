package logger

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"sale-service/config"
)

// New - Yangi logger yaratish
func New(cfg config.LoggerConfig) (*zap.Logger, error) {
	// Log level ni aniqlash
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// Encoder konfiguratsiyasi (JSON format)
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "vaqt",                                             // Vaqt maydoni
		LevelKey:       "daraja",                                           // Log daraja maydoni
		NameKey:        "logger",                                           // Logger nomi
		CallerKey:      "caller",                                           // Qaysi fayldan chaqirilgan
		FunctionKey:    zapcore.OmitKey,                                    // Function nomini ko'rsatmaslik
		MessageKey:     "xabar",                                            // Xabar maydoni
		StacktraceKey:  "stacktrace",                                       // Stacktrace maydoni
		LineEnding:     zapcore.DefaultLineEnding,                          // Qator oxiri
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,                   // Rang bilan level ko'rsatish
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"), // Vaqt formati
		EncodeDuration: zapcore.SecondsDurationEncoder,                     // Duration formati
		EncodeCaller:   zapcore.ShortCallerEncoder,                         // Qisqa caller nomi
	}

	// Terminal uchun encoder (rangli)
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// Fayl uchun encoder (JSON format)
	fileEncoderConfig := encoderConfig
	fileEncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder // Fayl uchun ranglar yo'q
	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

	// Fayl writer (log rotation bilan)
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.FilePath,   // Fayl manzili
		MaxSize:    cfg.MaxSize,    // Maksimal hajm (MB)
		MaxBackups: cfg.MaxBackups, // Nechta eski log saqlash
		MaxAge:     cfg.MaxAge,     // Necha kun saqlash
		Compress:   true,           // Eski loglarni siqish
	})

	// Bir nechta writer lar (terminal va fayl)
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level), // Terminal ga yozish
		zapcore.NewCore(fileEncoder, fileWriter, level),                    // Faylga yozish
	)

	// Logger yaratish
	logger := zap.New(core,
		zap.AddCaller(),                       // Qaysi fayldan chaqirilganini ko'rsatish
		zap.AddStacktrace(zapcore.ErrorLevel), // Error darajasida stacktrace ko'rsatish
	)

	return logger, nil
}

// GinLogger - Gin uchun middleware logger
func GinLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Request boshlanish vaqti
		start := time.Now()

		// Request ni bajarish
		c.Next()

		// Request tugash vaqti
		duration := time.Since(start)

		// Log yozish
		log.Info("HTTP Request",
			zap.String("method", c.Request.Method),          // HTTP metod
			zap.String("path", c.Request.URL.Path),          // URL path
			zap.Int("status", c.Writer.Status()),            // Status code
			zap.Duration("duration", duration),              // Qancha vaqt davom etdi
			zap.String("client_ip", c.ClientIP()),           // Client IP manzili
			zap.String("user_agent", c.Request.UserAgent()), // User agent
		)
	}
}
