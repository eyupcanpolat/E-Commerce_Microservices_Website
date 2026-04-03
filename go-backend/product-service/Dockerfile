FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY shared/ ./shared/
COPY product-service/ ./product-service/
WORKDIR /build/product-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/product-service ./cmd/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/product-service .
COPY --from=builder /build/product-service/data/ ./data/
RUN adduser -D -g '' appuser
USER appuser
EXPOSE 8082
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://localhost:8082/health || exit 1
CMD ["./product-service"]
