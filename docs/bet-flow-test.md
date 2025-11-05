# Testes de Fluxo de Aposta – Guia (via Makefile)

Este guia usa **somente alvos do Makefile** para subir a infra, rodar serviços e validar o fluxo:
**odds → ingest → Kafka → processor → cache/DB → wallet/bet → Kafka (bet_placed)**.

> **Pré-requisitos**
> - `docker` e `docker compose` instalados
> - `make` instalado
> - `docker-compose.yml` na **raiz do projeto**
> - `.env` com variáveis mínimas (veja `./.env.example`)

---

## 1) Subir infra base (compose)

```bash
make up
```
**Esperado:** containers `sbpp-postgres`, `sbpp-redis`, `sbpp-zookeeper`, `sbpp-kafka`, `sbpp-prometheus`, `sbpp-grafana` em **Up/Healthy** e topicos criados.

Verificar:
```bash
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

> Para desligar e limpar volumes: `make down`

---

## 2) Rodar serviços (via Makefile)

Abra **vários terminais** e rode os serviços abaixo:

### A) Odds Service (HTTP:8080 / Métricas:9095)
```bash
make odds
```
Health:
```bash
curl http://localhost:9095/healthz
# esperado: ok
```

### B) Supplier Simulator (WS em :8081/ws)
```bash
make supplier
```
**Esperado:** logs indicando WS em `ws://localhost:8081/ws` servindo stream.

### C) Ingest Service (conecta no WS, publica no Kafka) – Métricas:9096
```bash
make ingest
```
**Esperado:** log de criação/verificação do tópico `odds_updates` e envio de mensagens.

### D) Odds Processor Worker (consome Kafka, grava em cache/DB) – Métricas:9097
```bash
make processor
```
Health:
```bash
curl http://localhost:9097/healthz
# esperado: ok
```

> **Conferência opcional (Kafka):**
> ```bash
> docker exec -it sbpp-kafka kafka-console-consumer \
>   --bootstrap-server localhost:9092 \
>   --topic odds_updates --from-beginning
> # espere ver mensagens JSON de odds
> ```
>
> **Conferência opcional (DB):**
> ```bash
> docker exec -it sbpp-postgres psql -U bet -d bet_core \
>   -c "SELECT * FROM odds_current LIMIT 5;"
> docker exec -it sbpp-postgres psql -U bet -d bet_core \
>   -c "SELECT * FROM odds_history ORDER BY ts DESC LIMIT 5;"
> ```

---

## 3) Wallet & Bet (Sprint 3)

### A) Wallet Service (HTTP:8082 / Métricas:9098)
```bash
make wallet
```
Health:
```bash
curl http://localhost:9098/healthz
# esperado: ok
```

Provisionar/consultar carteira (cria on-demand):
```bash
curl "http://localhost:8082/wallet?userId=550e8400-e29b-41d4-a716-446655440000"
# esperado (exemplo):
# {
#   "user_id": "550e8400-e29b-41d4-a716-446655440000",
#   "balance_cents": 0,
#   "version": 1
# }
```

### B) Bet Service (HTTP:8083 / Métricas:9099)
```bash
make bet
```
Health:
```bash
curl http://localhost:9099/healthz
# esperado: ok
```

Criar aposta:
```bash
curl -s -X POST http://localhost:8083/bets \
 -H "Content-Type: application/json" \
 -d '{
   "userId":"550e8400-e29b-41d4-a716-446655440000",
   "eventId":"MATCH_001",
   "market":"MATCH_ODDS",
   "selection":"home",
   "stake_cents":1500,
   "odd_value":1.90
 }'
# esperado (exemplo):
# {
#   "betId": "d1f5a6b0-2ec1-4a0f-b6b1-4b6b2e7a8a00",
#   "status": "PENDING_CONFIRMATION",
#   "reservation_id": "rsv_12345"
# }
```

Conferir publicação no `bet_placed`:
```bash
docker exec -it sbpp-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic bet_placed --from-beginning
# esperado: JSON com userId, eventId, stake_cents, odd_value, etc.
```

> (Opcional) `GET /bets/{id}` no bet-service, se implementado, para ver status (PENDING_CONFIRMATION/CONFIRMED/REJECTED).

---

## 4) Métricas/Health (resumo)

- `odds-service` → `:9095/healthz` → `ok`
- `odds-ingest-service` → `:9096/healthz` → `ok`
- `odds-processor-worker` → `:9097/healthz` → `ok`
- `wallet-service` → `:9098/healthz` → `ok`
- `bet-service` → `:9099/healthz` → `ok`

> Se o Prometheus está configurado, ajuste os targets no `build/compose/prometheus/prometheus.yml` ou equivalente.

---

## 5) Troubleshooting rápido

- **Kafka `LEADER_NOT_AVAILABLE`** ao criar/consumir tópico: aguarde alguns segundos e rode de novo.
- **`password authentication failed for user` (Postgres)**: confira `POSTGRES_DSN` e credenciais do compose.
- **Algum `/healthz` falha**: suba o serviço correspondente com `make <alvo>` e verifique as portas / variáveis do `.env`.
- **WS supplier**: garanta `make supplier` antes de `make ingest`.
- **Prometheus marcando down**: atualize os targets para as portas de métricas corretas (9095..9099).

---

**Pronto.** Com isso você valida ponta a ponta com poucos comandos, tudo via `make`. 