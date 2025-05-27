# Dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/rpc-gateway .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/rpc-gateway .
COPY config.yaml .
EXPOSE 8545
EXPOSE 9090
CMD ["./rpc-gateway"]