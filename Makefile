SHELL := /bin/bash
ENV_FILE := .env
ifneq ("$(wildcard $(ENV_FILE))","")
	include $(ENV_FILE)
	export
endif

.PHONY: up down ps logs supplier odds ingest processor health topic dash fmt vet

## Infra
up:
	@docker compose up -d
	make topic
down:
	@docker compose down -v
ps:
	@docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
logs:
	@docker compose logs -f

## Kafka util (cria tópico se faltar)
topic:
	@echo "🔹 Criando tópico $(KAFKA_TOPIC_ODDS)..."
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server $(KAFKA_BROKER) \
	 --create --topic $(KAFKA_TOPIC_ODDS) --partitions 1 --replication-factor 1 || true
	@echo "🔹 Criando tópico $(KAFKA_TOPIC_BET_PLACED)..."
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server $(KAFKA_BROKER) \
	 --create --topic $(KAFKA_TOPIC_BET_PLACED) --partitions 1 --replication-factor 1 || true
	@echo "🔹 Criando tópico $(KAFKA_TOPIC_BET_CONFIRMED)..."
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server $(KAFKA_BROKER) \
	 --create --topic $(KAFKA_TOPIC_BET_CONFIRMED) --partitions 1 --replication-factor 1 || true
	@echo "✅ Tópicos verificados/criados!"

## Apps (rodar local)
odds:
	@ENV=$(ENV) SERVICE_NAME=$(SERVICE_NAME_ODDS) HTTP_PORT=$(HTTP_PORT) METRICS_PORT=$(METRICS_PORT) \
	 POSTGRES_DSN=$(POSTGRES_DSN) REDIS_ADDR=$(REDIS_ADDR) KAFKA_BROKERS=$(KAFKA_BROKERS) \
	 go run ./cmd/odds-service

ingest:
	@ENV=$(ENV) SUPPLIER_WS_URL=$(SUPPLIER_WS_URL) \
	 KAFKA_BROKERS=$(KAFKA_BROKERS) REDIS_ADDR=$(REDIS_ADDR) POSTGRES_DSN=$(POSTGRES_DSN) \
	 go run ./cmd/odds-ingest-service

processor:
	@ENV=$(ENV) POSTGRES_DSN=$(POSTGRES_DSN) REDIS_ADDR=$(REDIS_ADDR) KAFKA_BROKERS=$(KAFKA_BROKERS) \
	 go run ./cmd/odds-processor-worker

supplier:
	@ENV=$(ENV) go run ./cmd/supplier-simulator

wallet:
	@ENV=$(ENV) SERVICE_NAME=$(SERVICE_NAME_WALLET) HTTP_PORT=$(HTTP_PORT_WALLET) \
	METRICS_PORT=$(METRICS_PORT_WALLET) POSTGRES_DSN="$(POSTGRES_DSN)" \
	go run ./cmd/wallet-service

bet:
	@ENV=$(ENV) SERVICE_NAME=$(SERVICE_NAME_BET) HTTP_PORT=$(HTTP_PORT_BET) \
	METRICS_PORT=$(METRICS_PORT_BET) POSTGRES_DSN="$(POSTGRES_DSN)" \
	REDIS_ADDR="$(REDIS_ADDR)" KAFKA_BROKERS="$(KAFKA_BROKERS)" \
	go run ./cmd/bet-service

## Health quick-checks
health:
	@echo "== odds-ingest-service (9096) =="
	@curl -fsS http://localhost:9096/healthz || echo "ingest: DOWN"
	@echo
	@echo "== odds-processor-worker (9097) =="
	@curl -fsS http://localhost:9097/healthz || echo "processor: DOWN"
	@echo
	@echo "== odds-service (9095) =="
	@curl -fsS http://localhost:9095/healthz || echo "odds-service: DOWN"
	@echo
	@echo "== wallet-service (9098) =="
	@curl -fsS http://localhost:9098/healthz || echo "wallet-service: DOWN"
	@echo
	@echo "== bet-service (9099) =="
	@curl -fsS http://localhost:9099/healthz || echo "bet-service: DOWN"
	@echo
	@echo "== kafka topics =="
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server localhost:9092 --list || echo "kafka-cli: ERROR"

## DX
fmt:
	@go fmt ./...
vet:
	@go vet ./...

## Extras
dash:
	@echo "Grafana: http://localhost:3000  (admin/admin)"
	@echo "Prometheus: http://localhost:9090"


# Verifica .env
ifeq ("$(wildcard .env)","")
  $(warning ⚠️  Arquivo .env não encontrado. Copie .env.example para .env)
endif

# Exporta as variáveis do .env (linhas KEY=VALUE não comentadas)
ifneq ("$(wildcard .env)","")
  include .env
  export $(shell sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=.*/\1/p' .env)
endif

# Defaults de segurança (caso .env não esteja setado)
POSTGRES_DSN ?= postgres://bet:betpassword@localhost:5433/bet_core?sslmode=disable
REDIS_ADDR ?= localhost:6379
KAFKA_BROKERS ?= localhost:9092
SERVICE_NAME_ODDS ?= odds-service
HTTP_PORT ?= 8080
METRICS_PORT ?= 9095
SUPPLIER_WS_URL ?= ws://localhost:8081/ws
KAFKA_TOPIC_ODDS ?= odds_updates
REDIS_PUBSUB_CHANNEL ?= odds_updates_broadcast

print-env:
	@echo "ENV=$(ENV)"
	@echo "POSTGRES_DSN=$(POSTGRES_DSN)"
	@echo "REDIS_ADDR=$(REDIS_ADDR)"
	@echo "KAFKA_BROKERS=$(KAFKA_BROKERS)"
	@echo "SERVICE_NAME=$(SERVICE_NAME)"
	@echo "HTTP_PORT=$(HTTP_PORT)  METRICS_PORT=$(METRICS_PORT)"