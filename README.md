# sports-bet-platform-poc

Plataforma de apostas esportivas (POC) escrita em Go, inspirada em cenários reais de betting:
- ingestão e distribuição de odds em tempo real
- criação e confirmação de apostas
- controle de saldo do usuário (wallet) com consistência forte
- arquitetura orientada a eventos

## Serviços (planejados)

- `odds-service`: expõe odds/matches via REST e WebSocket
- `wallet-service`: gerencia saldo, débito/crédito e histórico financeiro
- `bet-service`: recebe apostas, reserva saldo e publica eventos `bet_placed`
- `bet-confirmation-worker`: consome `bet_placed`, confirma aposta com fornecedor e atualiza status
- `odds-ingest-service`: recebe stream do fornecedor externo e publica `odds_updates`
- `odds-processor-worker`: consome `odds_updates`, popula cache quente e banco de leitura
- `supplier-simulator`: simula um fornecedor externo (odds em tempo real e confirmação de apostas)
- `api-gateway`: camada de entrada (roteamento, auth, rate limit)

## Infra local

Tudo é rodado com Docker Compose:

- Postgres (armazenamento de carteira, apostas e leitura de odds)
- Redis (cache quente de odds/partidas)
- Kafka (barramento de eventos, ex: `odds_updates`, `bet_placed`)
- Prometheus + Grafana (observabilidade básica)
- (Zookeeper, requerido pelo Kafka)

### Subir infraestrutura

```bash
cd build/compose
docker compose up -d
