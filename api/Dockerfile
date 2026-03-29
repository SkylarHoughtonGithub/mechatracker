# syntax=docker/dockerfile:1

# ── Build stage ───────────────────────────────────────────────────────────────
# BUILDPLATFORM ensures this stage always runs natively on the host (arm64 on
# Apple Silicon). Go cross-compiles to TARGETOS/TARGETARCH without QEMU.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

# Populated automatically by buildx
ARG TARGETOS=linux
ARG TARGETARCH=arm64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Cross-compile natively — no emulation involved
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o bin/api ./cmd/api

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/bin/api .

EXPOSE 8080

ENTRYPOINT ["./api"]