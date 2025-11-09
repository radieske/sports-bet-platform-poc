FROM golang:1.23.12-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/cmd/api-gateway
RUN go build -o /api-gateway main.go

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /api-gateway ./api-gateway
COPY ./docs/swagger-ui /app/docs/swagger-ui
EXPOSE 8000
ENV HTTP_PORT=8000
CMD ["/app/api-gateway"]
