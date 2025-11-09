FROM golang:1.23.12-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/cmd/supplier-simulator
RUN go build -o /supplier-simulator main.go

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /supplier-simulator ./supplier-simulator
EXPOSE 8081 9094
ENV HTTP_PORT=8081 \
    METRICS_PORT=9094
CMD ["/app/supplier-simulator"]
