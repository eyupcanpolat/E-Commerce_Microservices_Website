FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY shared/ ./shared/
COPY address-service/ ./address-service/
WORKDIR /build/address-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/address-service ./cmd/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/address-service .

RUN adduser -D -g '' appuser
USER appuser
EXPOSE 8083
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://localhost:8083/health || exit 1
CMD ["./address-service"]
