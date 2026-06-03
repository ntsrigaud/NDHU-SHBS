# syntax=docker/dockerfile:1

# ─────────────────────────────────────────────────────────────────────────────
# Stage 1: builder
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Download dependencies first (layer-cached when go.mod/go.sum are unchanged).
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build both binaries.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o main   .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o migrate ./cmd/migrate

# ─────────────────────────────────────────────────────────────────────────────
# Stage 2: production
# ─────────────────────────────────────────────────────────────────────────────
FROM alpine:3.20 AS production

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && \
    adduser  -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /build/main    ./main
COPY --from=builder /build/migrate ./migrate
COPY --from=builder /build/migrations ./migrations

# Drop root before the process starts.
USER appuser

EXPOSE 8080

ENTRYPOINT ["./main"]
