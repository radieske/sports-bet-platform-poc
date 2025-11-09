# Testes de Fluxo WebSocket

Este documento descreve o fluxo de comunicação em tempo real entre os serviços da plataforma e orienta como testar o canal WebSocket de odds exposto pelo **odds-service** (via **API Gateway**).

---

## Visão Geral do Fluxo

1. **supplier-simulator** publica odds simuladas em tempo real.
2. **odds-ingest-service** consome essas odds e publica eventos no Kafka (`odds_updates`).
3. **odds-processor-worker** lê os eventos, atualiza o banco e envia atualizações ao Redis Pub/Sub.
4. **odds-service** (via WebSocket) transmite as atualizações aos clientes conectados.

O diagrama abaixo ilustra o fluxo de dados:

```
Supplier → Odds Ingest → Kafka (odds_updates) → Odds Processor → Redis Pub/Sub → Odds Service → WebSocket Clients
```

---

## Testando a Conexão WebSocket

O endpoint WebSocket está disponível em:

```
ws://localhost:8080/ws/odds
```

### Passos para Teste

1. Certifique-se de que o ambiente está rodando via Docker Compose:
   ```bash
   docker compose up -d
   ```
2. Utilize o `wscat` (ou ferramenta similar) para conectar-se ao canal:
   ```bash
   npx wscat -c ws://localhost:8080/ws/odds
   ```
3. Envie o seguinte payload para subscrever-se a um evento de teste:
   ```json
   { "type": "subscribe", "eventId": "MATCH_002" }
   ```
4. Aguarde mensagens automáticas de atualização de odds.  
   As respostas devem seguir o formato:
   ```json
   {
     "eventId": "MATCH_002",
     "market": "WINNER",
     "odd": 1.85,
     "timestamp": "2025-11-09T20:20:00Z"
   }
   ```

---


## Métricas e Monitoramento

Métricas relevantes para acompanhar o fluxo:

| Métrica | Descrição |
|----------|------------|
| `odds_proc_messages_consumed_total` | Mensagens Kafka processadas |
| `odds_proc_db_writes_total` | Escritas no banco de dados |
| `odds_proc_cache_sets_total` | Atualizações de cache Redis |
| `odds_proc_errors_total` | Erros de processamento |

As métricas podem ser consultadas em [http://localhost:9090](http://localhost:9090) via Prometheus.