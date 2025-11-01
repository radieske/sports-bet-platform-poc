# Sports Bet Platform POC

![Arquitetura Principal](https://raw.githubusercontent.com/radieske/sports-bet-platform-poc/main/docs/img/architecture-diagram.png)

> Plataforma de apostas esportivas construída em Go (Golang), utilizando arquitetura orientada a eventos e serviços desacoplados.

## Visão Geral

Esta aplicação é uma **prova de conceito (POC)** que simula o ecossistema de um site de apostas esportivas — cobrindo ingestão de odds de fornecedores externos, processamento em tempo real, cache distribuído e APIs de domínio independentes.

O foco é demonstrar boas práticas de **arquitetura distribuída**, **resiliência**, **observabilidade** e **organização por domínio (DDD)**.

---

## Stack Tecnológica

**Backend / Core**
- Go 1.23+
- gRPC e REST
- Kafka (event bus principal)
- Redis (cache quente)
- PostgreSQL (banco relacional RO/RW)
- Docker + Docker Compose
- Prometheus + Grafana (observabilidade)

**Infraestrutura**
- Kubernetes-ready design
- Configuração centralizada (pkg/config)
- Logs estruturados (Zap)
- Migrations automáticas (golang-migrate)
- Health checks e métricas Prometheus

---

## Estrutura de Diretórios (resumo)

```
├── build/compose
├── cmd/                  # executáveis (serviços)
│   ├── odds-ingest-service
│   ├── odds-processor-worker
│   ├── odds-service
│   └── supplier-simulator
├── docs/img/             # imagens de arquitetura
├── internal/
│   ├── infra/db/sql/pg/migrations  # V001__, V002__...
│   ├── odds-ingest
│   ├── odds-processor
│   ├── odds-service
│   └── shared            # cache, db, kafka, logger, metrics, config
└── pkg/contracts/events  # contratos cross-serviço (ex.: OddsUpdate)
```

---

## Serviços Atuais

| Serviço | Descrição |
|----------|------------|
| **supplier-simulator** | Simula o fornecedor externo enviando odds em WebSocket. |
| **odds-ingest-service** | Consome as odds do fornecedor, normaliza e publica no Kafka (`odds_updates`). |
| **odds-processor-worker** | Consome as odds do Kafka, atualiza Redis e persiste no Postgres RO. |
| **odds-service** | Exposição via REST + WebSocket para clientes, leitura de cache e fallback em DB. |

---

## Arquitetura e Fluxo de Dados

![Pipeline de Odds em Tempo Real](https://raw.githubusercontent.com/radieske/sports-bet-platform-poc/main/docs/img/real-time-odds-pipeline.png)

Fluxo end‑to‑end: **Supplier → Ingest → Kafka → Processor → Redis/Postgres → Odds Service (WS)**.

---

## Execução Local

### 1) Subir infraestrutura base
```bash
docker compose up -d
```

### 2) Rodar os serviços Go (atalhos)
```bash
make odds         # odds-service (REST + WS)
make processor    # odds-processor-worker (Kafka -> Redis/PG)
make ingest       # odds-ingest-service (WS supplier -> Kafka)
make supplier     # mock fornecedor WS
```

> Caso prefira rodar manualmente, veja as variáveis no `.env.example` e os targets do `Makefile`.

### 3) Testar WebSocket
- Tutorial: [`docs/ws-test.md`](docs/ws-test.md)

---

## Health & Metrics

- **Odds Service**: `http://localhost:9095/healthz` | `http://localhost:9095/metrics`  
- **Odds Processor**: `http://localhost:9097/healthz` | `http://localhost:9097/metrics`  
- **Ingest Service**: `http://localhost:9096/healthz` | `http://localhost:9096/metrics`  
- **Prometheus**: <http://localhost:9090>  
- **Grafana**: <http://localhost:3000> (admin/admin)

---

## Status Atual

- Infraestrutura base (Postgres, Redis, Kafka, Prometheus, Grafana)
- Migrations `V001__init_schema.sql`, `V002__odds_read_models.sql`
- Serviços `odds-ingest`, `odds-processor` e `odds-service` integrados e funcionais
- WebSocket público `/ws/odds` com subscribe/unsubscribe por `eventId`

---

## Próximos Passos 

- **Wallet Service** — saldo/ledger com concorrência protegida
- **Bet Service** — POST /bet, reserva de saldo e idempotência
- **Bet Confirmation Worker** — confirmação assíncrona no fornecedor
- Métricas custom e tracing distribuído