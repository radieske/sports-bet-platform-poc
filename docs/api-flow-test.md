# Testes de Fluxo de API e Apostas

Este documento descreve os principais testes REST realizados via **API Gateway** (`http://localhost:8000`), cobrindo os fluxos de consulta de odds, carteira e apostas.

---

## Fluxo Geral

```
API Gateway → Odds Service → Wallet Service → Bet Service → Kafka → Workers → Wallet Service (estorno)
```

Todos os endpoints seguem o padrão REST e estão acessíveis via Swagger em:
[http://localhost:8000/swagger/#/](http://localhost:8000/swagger/#/)

---

## 1. Consultas de Odds

### **GET /api/odds/v1/events**
Retorna a lista de eventos esportivos disponíveis para aposta.

**Request:**
```bash
curl -X 'GET'   'http://localhost:8000/api/odds/v1/events'   -H 'accept: application/json'
```

**Response:**
```json
[
  { "eventId": "MATCH_001", "homeTeam": "Flamengo", "awayTeam": "Palmeiras" },
  { "eventId": "MATCH_002", "homeTeam": "Grêmio", "awayTeam": "Internacional" },
  { "eventId": "MATCH_003", "homeTeam": "Corinthians", "awayTeam": "Santos" },
  { "eventId": "MATCH_004", "homeTeam": "São Paulo", "awayTeam": "Vasco" }
]
```

---

### **GET /api/odds/v1/events/{eventId}/markets**
Lista os mercados de apostas disponíveis para o evento informado.

**Request:**
```bash
curl -X 'GET'   'http://localhost:8000/api/odds/v1/events/MATCH_001/markets'   -H 'accept: application/json'
```

**Response:**
```json
[ { "market": "1x2" } ]
```

---

### **GET /api/odds/v1/events/{eventId}/odds**
Retorna as odds atuais do evento especificado.

**Request:**
```bash
curl -X 'GET'   'http://localhost:8000/api/odds/v1/events/MATCH_002/odds'   -H 'accept: application/json'
```

**Response:**
```json
[
  {
    "eventId": "MATCH_002",
    "market": "1x2",
    "homeOdd": 1.528,
    "drawOdd": 2.574,
    "awayOdd": 3.144,
    "version": 978,
    "updatedAt": "2025-11-09T21:07:45Z"
  }
]
```

---

## 2. Carteira (Wallet)

### **GET /api/wallet/wallet?userId={userId}**
Consulta o saldo atual da carteira de um usuário.

**Request:**
```bash
curl -X 'GET'   'http://localhost:8000/api/wallet/wallet?userId=USER_001'   -H 'accept: application/json'
```

**Response:**
```json
{
  "userId": "USER_001",
  "walletId": "437ff4e8-6e10-438c-8914-166f315cbcac",
  "balance_cents": 29000
}
```

---

### **POST /api/wallet/wallet/deposit**
Adiciona saldo à carteira de um usuário.  
Deve ser executado antes de criar apostas para garantir saldo suficiente.

**Request:**
```bash
curl -X 'POST'   'http://localhost:8000/api/wallet/wallet/deposit'   -H 'accept: application/json'   -H 'Content-Type: application/json'   -d '{
  "userId": "USER_001",
  "amount_cents": 30000,
  "external_ref": "string"
}'
```

**Response:**
```json
{
  "userId": "USER_001",
  "walletId": "437ff4e8-6e10-438c-8914-166f315cbcac",
  "balance_cents": 59000
}
```

---

## 3. Apostas (Bets)

### **POST /api/bets/bets**
Cria uma nova aposta.  
O valor apostado é debitado automaticamente do saldo da carteira e o evento é publicado no Kafka (`bet_placed`).

**Request:**
```bash
curl -X 'POST'   'http://localhost:8000/api/bets/bets'   -H 'accept: application/json'   -H 'Content-Type: application/json'   -d '{
  "userId": "USER_001",
  "eventId": "MATCH_002",
  "market": "1x2",
  "selection": "1",
  "stake_cents": 1000,
  "odd_value": 1.63
}'
```

**Response:**
```json
{
  "betId": "cfe1a384-bf05-410d-8137-6f280586bbd7",
  "status": "PENDING_CONFIRMATION"
}
```

---

### **GET /api/bets/bets/{betId}**
Consulta o status atual de uma aposta.

**Request:**
```bash
curl -X 'GET'   'http://localhost:8000/api/bets/bets/cfe1a384-bf05-410d-8137-6f280586bbd7'   -H 'accept: application/json'
```

**Response:**
```json
{
  "betId": "cfe1a384-bf05-410d-8137-6f280586bbd7",
  "status": "CONFIRMED"
}
```

---

## 4. Monitoramento e Falhas

### Healthcheck
Verifique a integridade do sistema:
```bash
curl http://localhost:8080/healthz
```

### Métricas
- Prometheus: [http://localhost:9090](http://localhost:9090)
- Grafana: [http://localhost:3000](http://localhost:3000) (admin/admin)

### Testes de Falha
1. **Parar o Kafka** e verificar logs de reconexão.  
2. **Interromper Postgres** e observar retries.  
3. **Forçar rejeição de aposta** via `supplier-simulator` para validar estorno.