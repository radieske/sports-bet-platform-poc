FROM golang:1.23.12-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/cmd/odds-service
RUN go build -o /odds-service main.go

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /odds-service ./odds-service
EXPOSE 8080 9095
ENV HTTP_PORT=8080 \
    METRICS_PORT=9095
CMD ["/app/odds-service"]
