# Arquitetura da Plataforma de Apostas Esportivas

![Arquitetura da Aplicação](https://raw.githubusercontent.com/radieske/sports-bet-platform-poc/main/docs/img/architecture-diagram.png)
*Diagrama geral da arquitetura da plataforma.*

---

## Visão Geral

A aplicação **Sports Bet Platform PoC** é composta por diversos serviços interconectados que simulam um ecossistema completo de apostas esportivas em tempo real.  
O objetivo principal é demonstrar um fluxo ponta a ponta — desde a atualização das odds pelo fornecedor até a confirmação da aposta e atualização do saldo do usuário.

A aplicação é totalmente containerizada e executada via **Docker Compose**.

---

## Estrutura de Serviços

| Serviço | Porta | Descrição |
|----------|--------|-----------|
| **API Gateway** | `8000` | Centraliza o acesso a todas as APIs REST (odds, wallet, bets) e expõe a documentação via Swagger. |
| **Supplier Simulator** | `8081` | Simula um fornecedor externo de odds e confirmações de apostas. |
| **Odds Ingest Service** | `8084` | Consome as odds do fornecedor e publica eventos no Kafka (`odds_updates`). |
| **Odds Processor Worker** | `8085` | Processa as odds recebidas, grava no banco e envia atualizações para o Redis (Pub/Sub). |
| **Odds Service** | `8080` | Exibe odds via API e WebSocket (`/ws/odds`). |
| **Wallet Service** | `8082` | Gerencia as carteiras dos usuários, incluindo depósitos, saques e estornos. |
| **Bet Service** | `8083` | Responsável pela criação e consulta de apostas. |
| **Bet Confirmation Worker** | `-` | Escuta o tópico `bet_placed` e confirma apostas via Supplier Simulator, publicando `bet_confirmed`. |
| **PostgreSQL** | `5432` | Armazena dados de apostas, odds e carteiras. |
| **Redis** | `6379` | Utilizado como cache e canal de publicação para WebSocket. |
| **Kafka + Zookeeper** | `9092 / 2181` | Responsáveis pelo fluxo assíncrono de eventos entre os microserviços. |
| **Prometheus** | `9090` | Coleta métricas dos serviços. |
| **Grafana** | `3000` | Visualiza as métricas coletadas pelo Prometheus. |

---

## Execução da Aplicação

A aplicação é executada **integralmente via Docker Compose**.

### 1. Subir o ambiente

```bash
docker compose up -d
```

Este comando inicializa todos os serviços da plataforma, incluindo o Kafka, Redis, Postgres e demais microserviços.

### 2. Verificar status dos serviços

```bash
docker compose ps
```

### 3. Logs gerais

```bash
docker compose logs -f
```

Para visualizar apenas erros:

```bash
docker compose logs -f | grep ERROR
```

### 4. Encerrar o ambiente

```bash
docker compose down
```

---

## Monitoramento e Observabilidade

| Ferramenta | URL | Descrição |
|-------------|-----|-----------|
| **Prometheus** | [http://localhost:9090](http://localhost:9090) | Métricas dos serviços e workers. |
| **Grafana** | [http://localhost:3000](http://localhost:3000) | Dashboards de observabilidade (login: admin / admin). |
| **Swagger** | [http://localhost:8000/swagger/#/](http://localhost:8000/swagger/#/) | Documentação das APIs disponíveis. |
| **Healthcheck** | `http://localhost:{porta}/healthz` | Endpoint de verificação de integridade para cada serviço. |

---

## Banco de Dados e Mensageria

### PostgreSQL
- Cada serviço possui schema e tabelas próprias.  
- O `odds-processor` e o `bet-service` persistem eventos transacionais.  
- Pode ser acessado via `psql`:
  ```bash
  docker exec -it sbpp-postgres psql -U bet -d bet_core
  ```

### Kafka
- Tópicos utilizados:
  - `odds_updates`
  - `bet_placed`
  - `bet_confirmed`
- Pode ser inspecionado com:
  ```bash
  docker exec -it sbpp-kafka kafka-topics --bootstrap-server kafka:9092 --list
  ```

### Redis
- Utilizado para cache e streaming de odds em tempo real.
- Conexão local:
  ```bash
  docker exec -it sbpp-redis redis-cli
  ```

---

## Testes e Validações

A plataforma inclui diferentes formas de validação do fluxo de ponta a ponta.

| Tipo | Descrição | Documentação |
|------|------------|---------------|
| **Testes de API REST** | Criação de apostas, depósitos, consulta de odds e saldo. | [docs/api-flow-test.md](./api-flow-test.md) |
| **Testes de WebSocket** | Recebimento em tempo real de atualizações de odds. | [docs/ws-test.md](./ws-test.md) |

---

## Conclusão

A arquitetura foi projetada para simular um ecossistema moderno de apostas esportivas, destacando:

- **Arquitetura distribuída baseada em eventos.**
- **Integração entre Kafka, Redis e PostgreSQL.**
- **Separação clara de responsabilidades entre microserviços.**
- **Monitoramento e métricas com Prometheus e Grafana.**

Este projeto serve como base para estudos, experimentação de integrações assíncronas e práticas de engenharia de software moderna.