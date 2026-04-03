# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy shared module first (dependency)
COPY shared/ ./shared/

# Copy service files
COPY cdn-service/ ./cdn-service/

# Build binary
WORKDIR /app/cdn-service
RUN go build -ldflags="-s -w" -o /cdn-service ./cmd/main.go

# ── Runtime stage ──────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /cdn-service .

# Default static directory — override with volume in docker-compose
RUN mkdir -p /app/static/images

EXPOSE 8085

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=5s \
  CMD wget -qO- http://localhost:8085/health || exit 1

CMD ["./cdn-service"]
