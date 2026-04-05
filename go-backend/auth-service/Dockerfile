# Auth Service Dockerfile
# Multi-stage build: builder stage compiles, final stage is minimal
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy shared module first (dependency)
COPY shared/ ./shared/

# Copy service source
COPY auth-service/ ./auth-service/

# Download dependencies and build
WORKDIR /build/auth-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/auth-service ./cmd/

# Final minimal image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/auth-service .


# Non-root user for security
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://localhost:8081/health || exit 1

CMD ["./auth-service"]
