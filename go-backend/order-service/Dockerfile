FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY shared/ ./shared/
COPY order-service/ ./order-service/
WORKDIR /build/order-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/order-service ./cmd/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/order-service .
COPY --from=builder /build/order-service/data/ ./data/
RUN adduser -D -g '' appuser
USER appuser
EXPOSE 8084
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://localhost:8084/health || exit 1
CMD ["./order-service"]
