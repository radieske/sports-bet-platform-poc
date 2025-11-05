package config

import (
	"os"

	ctopics "github.com/radieske/sports-bet-platform-poc/pkg/contracts/topics"
)

// Config centraliza variáveis de ambiente e parâmetros de execução dos serviços
// Inclui conexões, tópicos, canais, URLs e portas
type Config struct {
	Env         string // "local", "dev", "prod"
	ServiceName string // ex: "odds-service", "wallet-service", ...

	PostgresDSN  string
	RedisAddr    string
	KafkaBrokers string // "a:9092,b:9092"

	// Tópicos/canais
	TopicOddsUpdates     string
	TopicBetPlaced       string
	TopicBetConfirmed    string
	TopicBetPlacedDLQ    string
	TopicBetConfirmedDLQ string
	RedisPubSubChannel   string

	// Supplier mock
	SupplierWSURL string

	// Portas do serviço atual
	HTTPPort    string // Porta pública (ex.: API REST)
	MetricsPort string // Porta exclusiva para /metrics e /healthz
}

// Load carrega variáveis de ambiente e define defaults para cada serviço
// Resolve portas e tópicos conforme o SERVICE_NAME
func Load() Config {
	svc := getEnv("SERVICE_NAME", "")
	env := getEnv("ENV", "local")

	cfg := Config{
		Env:         env,
		ServiceName: svc,

		PostgresDSN:  getEnv("POSTGRES_DSN", "postgres://bet:betpassword@localhost:5433/bet_core?sslmode=disable"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),

		TopicOddsUpdates:     getEnv("KAFKA_TOPIC_ODDS", ctopics.OddsUpdates),
		TopicBetPlaced:       getEnv("KAFKA_TOPIC_BET_PLACED", ctopics.BetPlaced),
		TopicBetConfirmed:    getEnv("KAFKA_TOPIC_BET_CONFIRMED", ctopics.BetConfirmed),
		TopicBetPlacedDLQ:    getEnv("KAFKA_TOPIC_BET_PLACED_DLQ", ctopics.BetPlacedDLQ),
		TopicBetConfirmedDLQ: getEnv("KAFKA_TOPIC_BET_CONFIRMED_DLQ", ctopics.BetConfirmedDLQ),

		RedisPubSubChannel: getEnv("REDIS_PUBSUB_CHANNEL", "odds_updates_broadcast"),

		SupplierWSURL: getEnv("SUPPLIER_WS_URL", "ws://localhost:8081/ws"),
	}

	// Define portas padrão para cada serviço
	switch svc {
	case "wallet-service":
		cfg.HTTPPort = getEnv("HTTP_PORT_WALLET", "8082")
		cfg.MetricsPort = getEnv("METRICS_PORT_WALLET", "9098")
	case "bet-service":
		cfg.HTTPPort = getEnv("HTTP_PORT_BET", "8083")
		cfg.MetricsPort = getEnv("METRICS_PORT_BET", "9099")
	case "odds-ingest-service":
		cfg.HTTPPort = getEnv("HTTP_PORT_INGEST", "") // ingest não expõe HTTP público
		cfg.MetricsPort = getEnv("METRICS_PORT_INGEST", "9096")
	case "odds-processor-worker":
		cfg.HTTPPort = getEnv("HTTP_PORT_PROCESSOR", "")
		cfg.MetricsPort = getEnv("METRICS_PORT_PROCESSOR", "9097")
	case "odds-service":
		cfg.HTTPPort = getEnv("HTTP_PORT", "8080")
		cfg.MetricsPort = getEnv("METRICS_PORT", "9095")
	case "supplier-simulator":
		cfg.HTTPPort = getEnv("HTTP_PORT_SUPPLIER", "8081")
		cfg.MetricsPort = getEnv("METRICS_PORT_SUPPLIER", "9094")
	default:
		cfg.HTTPPort = getEnv("HTTP_PORT", "8080")
		cfg.MetricsPort = getEnv("METRICS_PORT", "9095")
	}

	return cfg
}

// getEnv retorna o valor da variável de ambiente ou o default
func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
