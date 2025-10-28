package config

import (
	"os"
)

type Config struct {
	ServiceName string

	HTTPPort    string // porta HTTP principal do serviço (ex: 8080)
	MetricsPort string // porta só para /metrics e /healthz

	PostgresDSN  string // conexão RW/RO
	RedisAddr    string // host:port
	KafkaBrokers string // "localhost:9092"

	Env string // "local", "dev", "prod"
}

func Load() Config {
	cfg := Config{
		ServiceName: getEnv("SERVICE_NAME", "odds-service"),

		HTTPPort:    getEnv("HTTP_PORT", "8080"),
		MetricsPort: getEnv("METRICS_PORT", "9095"),

		// porta externa 5433 mapeada para 5432 do container
		PostgresDSN:  getEnv("POSTGRES_DSN", "postgres://bet:betpassword@localhost:5433/bet_core?sslmode=disable"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),

		Env: getEnv("ENV", "local"),
	}
	return cfg
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
