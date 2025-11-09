package config

import (
    "os"
)

type Config struct {
    DatabaseHost     string
    DatabasePort     string
    DatabaseUser     string
    DatabasePassword string
    DatabaseName     string
    NatsURL          string
    NatsClusterID    string
    NatsClientID     string
    ServerPort       string
}

func Load() *Config {
    return &Config{
        DatabaseHost:     getEnv("DB_HOST", "127.0.0.1"),
        DatabasePort:     getEnv("DB_PORT", "5433"),
        DatabaseUser:     getEnv("DB_USER", "postgres"),
        DatabasePassword: getEnv("DB_PASSWORD", "121212"),
        DatabaseName:     getEnv("DB_NAME", "orders_db"),
        NatsURL:          getEnv("NATS_URL", "nats://localhost:4222"),
        NatsClusterID:    getEnv("NATS_CLUSTER_ID", "test-cluster"),
        NatsClientID:     getEnv("NATS_CLIENT_ID", "order-service-sub"),
        ServerPort:       getEnv("SERVER_PORT", "8080"),
    }
}

func getEnv(key, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return defaultValue
}