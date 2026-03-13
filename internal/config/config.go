package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config — конфигурация приложения (сервер, БД, Kafka, Redis).
type Config struct {
	Server ServerConfig `yaml:"server"`
	Redis  RedisConfig  `yaml:"redis"`
	DB     DBConfig     `yaml:"db"`
	Kafka  KafkaConfig  `yaml:"kafka"`
}

// ServerConfig — порт gRPC-сервера.
type ServerConfig struct {
	GRPCPort int `yaml:"grpc_port"`
}

// DBConfig — параметры подключения к PostgreSQL.
type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// KafkaConfig — брокеры и топик для событий платежей.
type KafkaConfig struct {
	TopicPaymentEvents string   `yaml:"topic_payment_events"`
	Brokers            []string `yaml:"brokers"`
}

// RedisConfig — адрес, пароль и TTL для кэша и идемпотентности.
type RedisConfig struct {
	BalanceTTL     time.Duration `yaml:"balance_ttl"`
	IdempotencyTTL time.Duration `yaml:"idempotency_ttl"`
	Addr           string        `yaml:"addr"`
	Password       string        `yaml:"password"`
}

// Load читает конфиг из YAML-файла и переопределяет значения из переменных окружения.
// Если path пустой, используются только значения по умолчанию и env.
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}

	overrideFromEnv(cfg)
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{GRPCPort: 50051},
		DB: DBConfig{
			Host:    "localhost",
			Port:    5432,
			User:    "postgres",
			DBName:  "payments",
			SSLMode: "disable",
		},
		Kafka: KafkaConfig{
			Brokers:            []string{"localhost:9092"},
			TopicPaymentEvents: "payment_events",
		},
		Redis: RedisConfig{
			Addr:           "localhost:6379",
			BalanceTTL:     5 * time.Minute,
			IdempotencyTTL: 24 * time.Hour,
		},
	}
}

func overrideFromEnv(cfg *Config) {
	if v := os.Getenv("GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.GRPCPort = p
		}
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.DB.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.DB.Port = p
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.DB.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.DB.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.DB.DBName = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		cfg.DB.SSLMode = v
	}
	if v := os.Getenv("KAFKA_BROKERS"); v != "" {
		cfg.Kafka.Brokers = strings.Split(v, ",")
		for i := range cfg.Kafka.Brokers {
			cfg.Kafka.Brokers[i] = strings.TrimSpace(cfg.Kafka.Brokers[i])
		}
	}
	if v := os.Getenv("KAFKA_TOPIC_PAYMENT_EVENTS"); v != "" {
		cfg.Kafka.TopicPaymentEvents = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_BALANCE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Redis.BalanceTTL = d
		}
	}
	if v := os.Getenv("REDIS_IDEM_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Redis.IdempotencyTTL = d
		}
	}
}

// DSN возвращает connection string для PostgreSQL (pgx).
func (c *DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
}
