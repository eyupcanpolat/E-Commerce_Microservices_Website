FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY shared/ ./shared/
COPY api-gateway/ ./api-gateway/
WORKDIR /build/api-gateway
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/api-gateway ./cmd/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/api-gateway .
RUN adduser -D -g '' appuser
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://localhost:8080/health || exit 1
CMD ["./api-gateway"]
