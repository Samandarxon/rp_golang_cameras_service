# Build stage
# FROM golang:1.21-alpine AS builder
FROM golang:alpine AS builder

# Keraku toollar
RUN apk add --no-cache git make

# Working directory
WORKDIR /app

# Dependencies ni yuklash
COPY go.mod go.sum ./
RUN go mod download

# Source code ni copy qilish
COPY . .

# Binary ni build qilish
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/sale-service cmd/main.go

# Runtime stage
FROM alpine:latest

# Security uchun non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Working directory
WORKDIR /app

# Keraku toollar
RUN apk --no-cache add ca-certificates

# Log papkasini yaratish va ruxsat berish
RUN mkdir -p /app/logs && chown appuser:appgroup /app/logs

# Binary ni copy qilish
COPY --from=builder /bin/sale-service .
COPY --from=builder /app/config/config.yaml ./config/

# User ni o'zgartirish
USER appuser

# Port
EXPOSE 8080 50051

# Command
CMD ["./sale-service"]

