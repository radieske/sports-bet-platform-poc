FROM golang:1.23.12-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/cmd/odds-processor-worker
RUN go build -o /odds-processor-worker main.go

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /odds-processor-worker ./odds-processor-worker
EXPOSE 8086 9097
ENV HTTP_PORT=8086 \
    METRICS_PORT=9097
CMD ["/app/odds-processor-worker"]
