# Teste do WebSocket de Odds

## Endpoints
- WS servidor: `ws://localhost:8080/ws/odds`
- Subscribe payload:
  ```json
  { "type": "subscribe", "eventId": "MATCH_001" }
  ```
- Unsubscribe:
  ```json
  { "type": "unsubscribe", "eventId": "MATCH_001" }
  ```

## Opção 1 — WebSocket King Client (GUI)
1. Abra https://websocketking.com
2. URL: `ws://localhost:8080/ws/odds` → **Connect**
3. Envie:
   ```json
   { "type": "subscribe", "eventId": "MATCH_001" }
   ```
4. Veja mensagens chegando em tempo real.

## Opção 2 — Postman
1. New → **WebSocket Request**
2. URL: `ws://localhost:8080/ws/odds` → **Connect**
3. Envie os JSONs acima.

## Opção 3 — CLI (`wscat`)
```bash
npm i -g wscat
wscat -c ws://localhost:8080/ws/odds
> {"type":"subscribe","eventId":"MATCH_001"}
```

## Verificações úteis
- Odds Service health:
  ```bash
  curl http://localhost:9095/healthz
  ```
- Processor health:
  ```bash
  curl http://localhost:9097/healthz
  ```
- Ingest health:
  ```bash
  curl http://localhost:9096/healthz
  ```

## Fluxo esperado
1. `make up` (infra)
2. `make topic` (se necessário)
3. `make processor`
4. `make odds`
5. `make seed` (simulador do fornecedor)
6. `make ingest` (publica no Kafka)
7. Conecte no WS e **subscribe** em `MATCH_001` → mensagens devem aparecer.