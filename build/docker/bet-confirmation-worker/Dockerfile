FROM golang:1.23.12-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/cmd/bet-confirmation-worker
RUN go build -o /bet-confirmation-worker main.go

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /bet-confirmation-worker ./bet-confirmation-worker
EXPOSE 8084 9098
ENV HTTP_PORT=8084 \
    METRICS_PORT=9098
CMD ["/app/bet-confirmation-worker"]
