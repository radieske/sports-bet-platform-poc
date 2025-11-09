# Sports Bet Platform PoC

![Arquitetura da aplicação](https://raw.githubusercontent.com/radieske/sports-bet-platform-poc/main/docs/img/architecture-diagram.png)
*Figura 1 - Arquitetura geral dos serviços e comunicação entre componentes.*

Este projeto demonstra uma arquitetura de microserviços voltada para uma plataforma de apostas esportivas. Ele integra serviços de ingestão, processamento e confirmação de apostas, cache Redis, banco de dados Postgres, mensageria Kafka e monitoramento com Prometheus e Grafana.

## Estrutura de Serviços

- **api-gateway**: expõe todas as APIs REST (odds, wallet e bet) via uma única interface HTTP.
- **supplier-simulator**: simula um fornecedor externo de odds e confirmações de apostas.
- **odds-ingest-service**: consome odds do fornecedor e publica no Kafka.
- **odds-processor-worker**: processa e persiste as odds no Postgres, cacheia no Redis e reenvia atualizações via Pub/Sub.
- **odds-service**: fornece dados e canal WebSocket para consulta de odds.
- **wallet-service**: gerencia saldos e estornos.
- **bet-service**: registra apostas e publica eventos no Kafka.
- **bet-confirmation-worker**: consome apostas criadas e confirma ou rejeita após resposta do fornecedor.
- **Postgres**, **Redis**, **Kafka**, **Prometheus** e **Grafana**: infraestrutura de apoio.

## Requisitos

- Docker e Docker Compose instalados.
- Portas 8080 (API Gateway), 9090+ e 3000 livres para uso local.

## Executando a aplicação

```bash
docker compose up -d --build
```

Verifique se todos os contêineres estão saudáveis:

```bash
docker compose ps
```

Para encerrar:

```bash
docker compose down
```

## Testes via Swagger

Toda a API REST está exposta através do **API Gateway**, disponível em:

**Swagger UI:** [http://localhost:8000/swagger/](http://localhost:8000/swagger/#/)

### Exemplos de testes

1. **Criar aposta** (`POST /bets`)
   - Envie um corpo JSON contendo `userId`, `eventId`, `stake_cents` e `odd_value`.
   - O evento `bet_placed` será publicado no Kafka e processado pelo `bet-confirmation-worker`.

2. **Consultar odds** (`GET /odds/{eventId}`)
   - Retorna as odds atuais do evento informando o `eventId` recebido via ingest.

3. **Consultar saldo da carteira** (`GET /wallet/{userId}`)
   - Exibe saldo atual de um usuário de teste.

## Testes diretos de serviços

### Healthchecks

Cada serviço expõe o endpoint `/healthz`:

```bash
curl http://localhost:8080/healthz   # api-gateway (proxy geral)
curl http://localhost:8082/healthz   # wallet-service
curl http://localhost:8083/healthz   # bet-service
```

O retorno esperado é `ok` com código HTTP 200.

### Teste de conexão WebSocket

O `odds-service` fornece um canal WebSocket acessível em:

```
ws://localhost:8080/ws/odds
```

Você pode testar com o comando `wscat` ou usando uma extensão de navegador:

```bash
npx wscat -c ws://localhost:8080/ws/odds
```

Após conectado, envie o payload abaixo para subscrever-se a um evento de teste:

```json
{ "type": "subscribe", "eventId": "MATCH_002" }
```

Se as odds estiverem sendo publicadas, você receberá mensagens automáticas com atualizações em tempo real.

### Prometheus e Grafana

- **Prometheus:** [http://localhost:9090](http://localhost:9090)
- **Grafana:** [http://localhost:3000](http://localhost:3000) (login padrão: `admin` / `admin`)

O Grafana já vem configurado para consumir métricas dos serviços via Prometheus. Você pode criar dashboards customizados para:

- Mensagens consumidas do Kafka (`*_messages_consumed_total`)
- Escritas no banco (`*_db_writes_total`)
- Erros (`*_errors_total`)
- Cache hits/sets (`*_cache_sets_total`)

## Logs

Para acompanhar logs de todos os serviços em tempo real:

```bash
docker compose logs -f
```

Ou apenas de um serviço específico:

```bash
docker compose logs -f sbpp-bet-service
```

Filtrar apenas erros:

```bash
docker compose logs -f | grep -i "error"
```

## Tópicos Kafka relevantes

| Tópico | Produzido por | Consumido por |
|----------|----------------|----------------|
| `odds_updates` | odds-ingest-service | odds-processor-worker |
| `bet_placed` | bet-service | bet-confirmation-worker |
| `bet_confirmed` | bet-confirmation-worker | wallet-service (para futuras integrações) |

## Encerrando e limpando dados

```bash
docker compose down -v
```

Esse comando remove volumes e zera o estado do banco, cache e mensageria.

---

Este repositório tem finalidade educacional e demonstra princípios de arquitetura orientada a eventos, processamento assíncrono e monitoramento de serviços distribuídos.