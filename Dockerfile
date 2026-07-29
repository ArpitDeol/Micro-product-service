# =========================
# Build Stage
# =========================
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Required for downloading Go dependencies
RUN apk add --no-cache git ca-certificates

# Copy dependency files first for better Docker cache
COPY go.mod go.sum ./

RUN go mod download

# Copy application source code
COPY . .

# Build a static Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o product-service \
    ./main.go


# =========================
# Runtime Stage
# =========================
FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates curl \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

COPY --from=builder /app/product-service /app/product-service

ENV PORT=8003

EXPOSE 8003

USER appuser

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl --fail http://localhost:8003/health || exit 1

CMD ["/app/product-service"]