SHELL := /bin/bash
ENV_FILE := .env
ifneq ("$(wildcard $(ENV_FILE))","")
	include $(ENV_FILE)
	export
endif

.PHONY: up down ps logs seed odds ingest processor health topic dash fmt vet

## Infra
up:
	@docker compose up -d
down:
	@docker compose down -v
ps:
	@docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
logs:
	@docker compose logs -f

## Kafka util (cria tópico se faltar)
topic:
	@docker exec -it sbpp-kafka kafka-topics --bootstrap-server localhost:9092 \
	 --create --topic $(KAFKA_TOPIC_ODDS) --partitions 1 --replication-factor 1 || true

## Apps (rodar local)
odds:
	@ENV=$(ENV) SERVICE_NAME=$(SERVICE_NAME) HTTP_PORT=$(HTTP_PORT) METRICS_PORT=$(METRICS_PORT) \
	 POSTGRES_DSN=$(POSTGRES_DSN) REDIS_ADDR=$(REDIS_ADDR) KAFKA_BROKERS=$(KAFKA_BROKERS) \
	 go run ./cmd/odds-service

ingest:
	@ENV=$(ENV) SUPPLIER_WS_URL=$(SUPPLIER_WS_URL) \
	 KAFKA_BROKERS=$(KAFKA_BROKERS) REDIS_ADDR=$(REDIS_ADDR) POSTGRES_DSN=$(POSTGRES_DSN) \
	 go run ./cmd/odds-ingest-service

processor:
	@ENV=$(ENV) POSTGRES_DSN=$(POSTGRES_DSN) REDIS_ADDR=$(REDIS_ADDR) KAFKA_BROKERS=$(KAFKA_BROKERS) \
	 go run ./cmd/odds-processor-worker

seed:
	@ENV=$(ENV) go run ./cmd/supplier-simulator

## Health quick-checks
health:
	@echo "odds-service:"
	@curl -sS http://localhost:$(METRICS_PORT)/healthz || true
	@echo ""
	@echo "processor:"
	@curl -sS http://localhost:9097/healthz || true
	@echo ""
	@echo "ingest:"
	@curl -sS http://localhost:9096/healthz || true

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
SERVICE_NAME ?= odds-service
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