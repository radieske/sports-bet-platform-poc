# Testes de API – Fluxo de Carteira e Apostas

Este documento descreve o procedimento para validar o funcionamento dos serviços relacionados fluxo de carteira e apostas da plataforma de apostas esportivas.

---

## 1. Serviços necessários

Para que os testes funcionem corretamente, é necessário subir toda a infraestrutura e executar os serviços principais na seguinte ordem:

```bash
make up
make supplier
make odds
make processor
make ingest
make wallet
make bet
```

Esses comandos sobem:

- Infraestrutura base (`Postgres`, `Redis`, `Kafka`, `Prometheus`, `Grafana`)
- Serviços core:
  - supplier-simulator (emite eventos e odds em tempo real)
  - odds-service (consulta e WebSocket de odds)
  - odds-ingest-service (consome fornecedor e publica no Kafka)
  - odds-processor-worker (atualiza cache e banco)
  - wallet-service (gerencia carteira e saldo)
  - bet-service (cria apostas e publica eventos)

Verifique a saúde de todos os serviços com:

```bash
make health
```

Todos devem retornar `ok` em `/healthz`.

---

## 2. Testes de API

### 2.1 Criar ou consultar carteira

```bash
curl -s "http://localhost:8082/wallet?userId=USER_001"
```

Resposta esperada:
```json
{
  "user_id": "USER_001",
  "balance_cents": 0,
  "version": 1
}
```

---

### 2.2 Depositar saldo

```bash
curl -s -X POST http://localhost:8082/wallet/deposit  -H "Content-Type: application/json"  -d '{"userId":"USER_001","amount_cents":10000,"external_ref":"dep-1"}'
```

Resposta esperada:
```json
{
  "user_id": "USER_001",
  "balance_cents": 10000,
  "version": 2
}
```

---

### 2.3 Criar aposta

```bash
curl -s -X POST http://localhost:8083/bets  -H "Content-Type: application/json"  -d '{"userId":"USER_001","eventId":"MATCH_001","market":"MATCH_ODDS","selection":"home","stake_cents":1500,"odd_value":1.90}'
```

Resposta esperada:
```json
{
  "betId": "d1f5a6b0-2ec1-4a0f-b6b1-4b6b2e7a8a00",
  "status": "PENDING_CONFIRMATION",
  "reservation_id": "rsv_12345"
}
```

---

### 2.4 Consultar aposta

Substitua `<BET_ID>` pelo ID retornado na etapa anterior.

```bash
curl -s http://localhost:8083/bets/<BET_ID>
```

Resposta esperada:
```json
{
  "betId": "d1f5a6b0-2ec1-4a0f-b6b1-4b6b2e7a8a00",
  "user_id": "USER_001",
  "event_id": "MATCH_001",
  "status": "PENDING_CONFIRMATION",
  "stake_cents": 1500,
  "odd_value": 1.9
}
```

---

### 2.5 Validar publicação no Kafka

Para garantir que o evento da aposta foi publicado corretamente:

```bash
docker exec -it sbpp-kafka kafka-console-consumer   --bootstrap-server localhost:9092   --topic bet_placed --from-beginning
```

Esperado: mensagens JSON com `userId`, `eventId`, `stake_cents`, `odd_value`, etc.

---

## 3. Observações

- As portas podem variar conforme o `.env`.
- Certifique-se de que os tópicos Kafka foram criados com `make topic` antes de rodar os serviços.
- Caso o Prometheus esteja configurado, as métricas de cada serviço estarão disponíveis em:
  - `:9095/metrics` (odds-service)
  - `:9096/metrics` (odds-ingest-service)
  - `:9097/metrics` (odds-processor-worker)
  - `:9098/metrics` (wallet-service)
  - `:9099/metrics` (bet-service)

---

Status: Testes executados com sucesso.