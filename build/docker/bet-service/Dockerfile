FROM golang:1.23.12-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/cmd/bet-service
RUN go build -o /bet-service main.go

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /bet-service ./bet-service
EXPOSE 8083 9099
ENV HTTP_PORT=8083 \
    METRICS_PORT=9099
CMD ["/app/bet-service"]
