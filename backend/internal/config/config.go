package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Ditto    DittoConfig    `mapstructure:"ditto"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
	Environment  string `mapstructure:"environment"`
}

// DatabaseConfig holds database-specific configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
	TimeZone string `mapstructure:"timezone"`
}

// DittoConfig holds Eclipse Ditto API configuration
type DittoConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	APIToken string `mapstructure:"api_token"`
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers        string `mapstructure:"brokers"`
	ConsumerGroup  string `mapstructure:"consumer_group"`
	SecurityEnable bool   `mapstructure:"security_enable"`
	SecurityUser   string `mapstructure:"security_user"`
	SecurityPass   string `mapstructure:"security_pass"`
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	Secret                 string `mapstructure:"secret"`
	ExpirationHours        int    `mapstructure:"expiration_hours"`
	RefreshSecret          string `mapstructure:"refresh_secret"`
	RefreshExpirationHours int    `mapstructure:"refresh_expiration_hours"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

// LoadConfig loads the application configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	var config Config

	// Set default configuration file path if not provided
	if configPath == "" {
		configPath = "./config"
	}

	// Initialize Viper
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)

	// Set environment variable prefix for overrides
	v.SetEnvPrefix("DIGITAL_EGIZ")

	// Set environment variable separator for nested structs
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read configuration from file
	if err := v.ReadInConfig(); err != nil {
		// If the configuration file is not found, that's fine, we'll use defaults and env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	// Set up environment variable binding
	v.AutomaticEnv()

	// Set defaults
	setDefaults(v)

	// Unmarshal configuration
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults sets default values for the configuration
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", 15)  // seconds
	v.SetDefault("server.write_timeout", 15) // seconds
	v.SetDefault("server.idle_timeout", 60)  // seconds
	v.SetDefault("server.environment", "development")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.dbname", "digital_egiz")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.timezone", "UTC")

	// Ditto defaults
	v.SetDefault("ditto.url", "http://ditto:8080")

	// Kafka defaults
	v.SetDefault("kafka.brokers", "kafka:9092")
	v.SetDefault("kafka.consumer_group", "digital-egiz")
	v.SetDefault("kafka.security_enable", false)

	// JWT defaults
	v.SetDefault("jwt.expiration_hours", 24)
	v.SetDefault("jwt.refresh_expiration_hours", 168) // 7 days

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output_path", "stdout")
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate JWT secrets are set
	if config.JWT.Secret == "" {
		// In development mode, set a default secret
		if config.Server.Environment == "development" {
			config.JWT.Secret = "development-jwt-secret-key-change-in-production"
		} else {
			return fmt.Errorf("JWT secret is required in non-development environments")
		}
	}

	if config.JWT.RefreshSecret == "" {
		// In development mode, set a default refresh secret
		if config.Server.Environment == "development" {
			config.JWT.RefreshSecret = "development-refresh-secret-key-change-in-production"
		} else {
			return fmt.Errorf("JWT refresh secret is required in non-development environments")
		}
	}

	// Validate database password is set
	if config.Database.Password == "" {
		// Check if it's available in environment variable
		dbPassword := os.Getenv("DIGITAL_EGIZ_DATABASE_PASSWORD")
		if dbPassword == "" {
			if config.Server.Environment != "development" {
				return fmt.Errorf("database password is required in non-development environments")
			}
		} else {
			config.Database.Password = dbPassword
		}
	}

	return nil
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode, c.TimeZone)
}

// IsProduction returns true if the environment is production
func (c *ServerConfig) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if the environment is development
func (c *ServerConfig) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsTest returns true if the environment is test
func (c *ServerConfig) IsTest() bool {
	return c.Environment == "test"
}
