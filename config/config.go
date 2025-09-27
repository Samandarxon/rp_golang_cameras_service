package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config - Barcha konfiguratsiya sozlamalari
type Config struct {
	ServiceName string `mapstructure:"service_name"` // Servis nomi
	Version     string `mapstructure:"version"`      // Servis versiyasi
	Environment string `mapstructure:"environment"`  // dev, staging, production

	HTTP     HTTPConfig     `mapstructure:"http"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Logger   LoggerConfig   `mapstructure:"logger"`
}

// HTTPConfig - HTTP server sozlamalari
type HTTPConfig struct {
	Port         string        `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// GRPCConfig - gRPC server sozlamalari
type GRPCConfig struct {
	Port string `mapstructure:"port"`
}

// DatabaseConfig - PostgreSQL sozlamalari
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            string `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"db_name"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`    // Maksimal ochiq connection lar
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`    // Maksimal bo'sh connection lar
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // Connection ning maksimal umri (daqiqa)
}

// RedisConfig - Redis sozlamalari
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// KafkaConfig - Kafka sozlamalari
type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`  // Kafka broker larning manzillari
	Topic   string   `mapstructure:"topic"`    // Qaysi topic dan xabar olish
	GroupID string   `mapstructure:"group_id"` // Consumer group ID
}

// LoggerConfig - Logger sozlamalari
type LoggerConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	FilePath   string `mapstructure:"file_path"`   // Log faylning manzili
	MaxSize    int    `mapstructure:"max_size"`    // Fayl hajmi (MB)
	MaxBackups int    `mapstructure:"max_backups"` // Nechta eski log saqlash
	MaxAge     int    `mapstructure:"max_age"`     // Necha kun saqlash
}

// Load - config.yaml faylidan konfiguratsiyani yuklash
func Load() (*Config, error) {
	viper.SetConfigName("config")   // config.yaml
	viper.SetConfigType("yaml")     // YAML format
	viper.AddConfigPath("./config") // config papkasida qidirish
	viper.AddConfigPath(".")        // Ildiz papkada qidirish

	// Environment o'zgaruvchilarini ham o'qish
	viper.AutomaticEnv()

	// Default qiymatlar
	setDefaults()

	// Konfiguratsiya faylini o'qish
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Config fayl topilmadi, environmentdan foydalanilmoqda: %v\n", err)
		return nil, err
	}

	var cfg Config
	// YAML dan struct ga o'tkazish
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Environment variable lar ustunligi
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		cfg.Kafka.Brokers = []string{brokers}
	}
	if topic := os.Getenv("KAFKA_TOPIC"); topic != "" {
		cfg.Kafka.Topic = topic
	}
	if groupID := os.Getenv("KAFKA_GROUP_ID"); groupID != "" {
		cfg.Kafka.GroupID = groupID
	}

	fmt.Printf("Kafka Brokers: %v\n", cfg.Kafka.Brokers)
	fmt.Printf("Kafka Topic: %s\n", cfg.Kafka.Topic)

	return &cfg, nil
}

// setDefaults - Default sozlamalar
func setDefaults() {
	viper.SetDefault("service_name", "sale-service")
	viper.SetDefault("version", "1.0.0")
	viper.SetDefault("environment", "development")

	viper.SetDefault("http.port", ":8080")
	viper.SetDefault("grpc.port", ":50051")

	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", 5)
}
