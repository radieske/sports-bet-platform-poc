FROM golang:1.23.12-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/cmd/odds-ingest-service
RUN go build -o /odds-ingest-service main.go

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /odds-ingest-service ./odds-ingest-service
EXPOSE 8085 9096
ENV HTTP_PORT=8085 \
    METRICS_PORT=9096
CMD ["/app/odds-ingest-service"]
