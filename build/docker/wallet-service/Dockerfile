FROM golang:1.23.12-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/cmd/wallet-service
RUN go build -o /wallet-service main.go

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /wallet-service ./wallet-service
EXPOSE 8082 9098
ENV HTTP_PORT=8082 \
    METRICS_PORT=9098
CMD ["/app/wallet-service"]
