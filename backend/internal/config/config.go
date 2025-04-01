package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Ditto    DittoConfig
	Kafka    KafkaConfig
	JWT      JWTConfig
	Logging  LoggingConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
}

// DatabaseConfig holds database connection information
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DittoConfig holds Eclipse Ditto configuration
type DittoConfig struct {
	URL      string
	Username string
	Password string
	APIToken string
}

// KafkaConfig holds Kafka connection configuration
type KafkaConfig struct {
	Brokers        string
	ConsumerGroup  string
	Topics         []string
	SecurityEnable bool
	SecurityUser   string
	SecurityPass   string
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	Secret        string
	ExpirationSec int
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string
	Production bool
}

// LoadConfig loads application configuration from file and environment
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default configuration values
	setDefaults(v)

	// Load config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Override with environment variables
	v.SetEnvPrefix("DIGITAL_EGIZ")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Create config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.readTimeout", 10)
	v.SetDefault("server.writeTimeout", 10)
	v.SetDefault("server.idleTimeout", 60)

	// Database defaults
	v.SetDefault("database.host", "postgres")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.dbname", "digital_egiz")
	v.SetDefault("database.sslmode", "disable")

	// Ditto defaults
	v.SetDefault("ditto.url", "http://ditto:8080")
	v.SetDefault("ditto.username", "ditto")
	v.SetDefault("ditto.password", "ditto")

	// Kafka defaults
	v.SetDefault("kafka.brokers", "kafka:9092")
	v.SetDefault("kafka.consumerGroup", "digital-egiz-backend")
	v.SetDefault("kafka.topics", []string{
		"ditto.twin.events.v1",
		"ml.output.v1.anomaly",
	})
	v.SetDefault("kafka.securityEnable", false)

	// JWT defaults
	v.SetDefault("jwt.expirationSec", 86400) // 24 hours

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.production", false)
} 