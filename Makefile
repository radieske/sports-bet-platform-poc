SHELL := /bin/bash
ENV_FILE := .env
ifneq ("$(wildcard $(ENV_FILE))","")
	include $(ENV_FILE)
	export
endif

.PHONY: up down ps logs odds ingest processor supplier wallet bet confirm-worker gateway topic topic-list consume-odds consume-bet-placed consume-bet-confirmed health fmt vet dash print-env

POSTGRES_DSN ?= postgres://bet:betpassword@localhost:5433/bet_core?sslmode=disable
REDIS_ADDR ?= localhost:6379
KAFKA_BROKERS ?= localhost:9092
KAFKA_BROKER ?= localhost:9092

SERVICE_NAME_ODDS ?= odds-service
SERVICE_NAME_WALLET ?= wallet-service
SERVICE_NAME_BET ?= bet-service
SERVICE_NAME_INGEST ?= odds-ingest-service
SERVICE_NAME_PROCESSOR ?= odds-processor-worker
SERVICE_NAME_SUPPLIER ?= supplier-simulator

HTTP_PORT_ODDS ?= 8080
METRICS_PORT_ODDS ?= 9095
HTTP_PORT_WALLET ?= 8082
METRICS_PORT_WALLET ?= 9098
HTTP_PORT_BET ?= 8083
METRICS_PORT_BET ?= 9099
HTTP_PORT_SUPPLIER ?= 8081
METRICS_PORT_SUPPLIER ?= 9094
METRICS_PORT_INGEST ?= 9096
METRICS_PORT_PROCESSOR ?= 9097
HTTP_PORT_GATEWAY ?= 8000

SUPPLIER_WS_URL ?= ws://localhost:8081/ws

KAFKA_TOPIC_ODDS ?= odds_updates
KAFKA_TOPIC_BET_PLACED ?= bet_placed
KAFKA_TOPIC_BET_CONFIRMED ?= bet_confirmed
KAFKA_TOPIC_BET_PLACED_DLQ ?= bet_placed_dlq
KAFKA_TOPIC_BET_CONFIRMED_DLQ ?= bet_confirmed_dlq

REDIS_PUBSUB_CHANNEL ?= odds_updates_broadcast

up:
	@docker compose up -d
	make topic

down:
	@docker compose down -v

ps:
	@docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'

logs:
	@docker compose logs -f

topic:
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server $(KAFKA_BROKER) --create --topic $(KAFKA_TOPIC_ODDS) --partitions 1 --replication-factor 1 || true
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server $(KAFKA_BROKER) --create --topic $(KAFKA_TOPIC_BET_PLACED) --partitions 1 --replication-factor 1 || true
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server $(KAFKA_BROKER) --create --topic $(KAFKA_TOPIC_BET_CONFIRMED) --partitions 1 --replication-factor 1 || true
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server $(KAFKA_BROKER) --create --topic $(KAFKA_TOPIC_BET_PLACED_DLQ) --partitions 1 --replication-factor 1 || true
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server $(KAFKA_BROKER) --create --topic $(KAFKA_TOPIC_BET_CONFIRMED_DLQ) --partitions 1 --replication-factor 1 || true

topic-list:
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server $(KAFKA_BROKER) --list

consume-odds:
	@docker exec -it sbpp-kafka kafka-console-consumer --bootstrap-server $(KAFKA_BROKER) --topic $(KAFKA_TOPIC_ODDS) --from-beginning

consume-bet-placed:
	@docker exec -it sbpp-kafka kafka-console-consumer --bootstrap-server $(KAFKA_BROKER) --topic $(KAFKA_TOPIC_BET_PLACED) --from-beginning

consume-bet-confirmed:
	@docker exec -it sbpp-kafka kafka-console-consumer --bootstrap-server $(KAFKA_BROKER) --topic $(KAFKA_TOPIC_BET_CONFIRMED) --from-beginning

odds:
	@ENV=$(ENV) SERVICE_NAME=$(SERVICE_NAME_ODDS) HTTP_PORT=$(HTTP_PORT_ODDS) METRICS_PORT=$(METRICS_PORT_ODDS) POSTGRES_DSN="$(POSTGRES_DSN)" REDIS_ADDR="$(REDIS_ADDR)" KAFKA_BROKERS="$(KAFKA_BROKERS)" REDIS_PUBSUB_CHANNEL="$(REDIS_PUBSUB_CHANNEL)" go run ./cmd/odds-service

ingest:
	@ENV=$(ENV) SERVICE_NAME=$(SERVICE_NAME_INGEST) KAFKA_BROKERS="$(KAFKA_BROKERS)" SUPPLIER_WS_URL="$(SUPPLIER_WS_URL)" METRICS_PORT=$(METRICS_PORT_INGEST) go run ./cmd/odds-ingest-service

processor:
	@ENV=$(ENV) SERVICE_NAME=$(SERVICE_NAME_PROCESSOR) POSTGRES_DSN="$(POSTGRES_DSN)" REDIS_ADDR="$(REDIS_ADDR)" KAFKA_BROKERS="$(KAFKA_BROKERS)" METRICS_PORT=$(METRICS_PORT_PROCESSOR) go run ./cmd/odds-processor-worker

supplier:
	@ENV=$(ENV) SERVICE_NAME=$(SERVICE_NAME_SUPPLIER) HTTP_PORT=$(HTTP_PORT_SUPPLIER) METRICS_PORT=$(METRICS_PORT_SUPPLIER) go run ./cmd/supplier-simulator

wallet:
	@ENV=$(ENV) SERVICE_NAME=$(SERVICE_NAME_WALLET) HTTP_PORT=$(HTTP_PORT_WALLET) METRICS_PORT=$(METRICS_PORT_WALLET) POSTGRES_DSN="$(POSTGRES_DSN)" REDIS_ADDR="$(REDIS_ADDR)" KAFKA_BROKERS="$(KAFKA_BROKERS)" go run ./cmd/wallet-service

bet:
	@ENV=$(ENV) SERVICE_NAME=$(SERVICE_NAME_BET) HTTP_PORT=$(HTTP_PORT_BET) METRICS_PORT=$(METRICS_PORT_BET) POSTGRES_DSN="$(POSTGRES_DSN)" REDIS_ADDR="$(REDIS_ADDR)" KAFKA_BROKERS="$(KAFKA_BROKERS)" go run ./cmd/bet-service

confirm-worker:
	@ENV=$(ENV) SERVICE_NAME=bet-confirmation-worker POSTGRES_DSN="$(POSTGRES_DSN)" REDIS_ADDR="$(REDIS_ADDR)" KAFKA_BROKERS="$(KAFKA_BROKERS)" METRICS_PORT=9100 go run ./cmd/bet-confirmation-worker

gateway:
	@ENV=$(ENV) SERVICE_NAME=api-gateway HTTP_PORT=$(HTTP_PORT_GATEWAY) go run ./cmd/api-gateway

health:
	@curl -fsS http://localhost:$(METRICS_PORT_INGEST)/healthz || echo "ingest: DOWN"
	@curl -fsS http://localhost:$(METRICS_PORT_PROCESSOR)/healthz || echo "processor: DOWN"
	@curl -fsS http://localhost:$(METRICS_PORT_ODDS)/healthz || echo "odds-service: DOWN"
	@curl -fsS http://localhost:$(METRICS_PORT_WALLET)/healthz || echo "wallet-service: DOWN"
	@curl -fsS http://localhost:$(METRICS_PORT_BET)/healthz || echo "bet-service: DOWN"
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server $(KAFKA_BROKER) --list || echo "kafka-cli: ERROR"

fmt:
	@go fmt ./...

vet:
	@go vet ./...

dash:
	@echo "Grafana: http://localhost:3000"
	@echo "Prometheus: http://localhost:9090"

print-env:
	@env | grep -E 'ENV|POSTGRES|REDIS|KAFKA|SERVICE|HTTP_PORT|METRICS_PORT|SUPPLIER_WS_URL'