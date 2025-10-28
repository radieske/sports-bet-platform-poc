package config

import (
	"os"
)

type Config struct {
	ServiceName  string
	HTTPPort     string
	PostgresDSN  string
	RedisAddr    string
	KafkaBrokers string
	Env          string
}

func Load() Config {
	cfg := Config{
		ServiceName:  getEnv("SERVICE_NAME", "unknown-service"),
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
		PostgresDSN:  getEnv("POSTGRES_DSN", "postgres://bet:betpassword@localhost:5433/bet_core?sslmode=disable"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		Env:          getEnv("ENV", "local"),
	}
	return cfg
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
