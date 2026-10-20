# syntax=docker/dockerfile:1

# ── Build stage ───────────────────────────────────────────────────────────────
# BUILDPLATFORM ensures this stage always runs natively on the host (arm64 on
# Apple Silicon). Go cross-compiles to TARGETOS/TARGETARCH without QEMU.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

# Populated automatically by buildx
ARG TARGETOS=linux
ARG TARGETARCH=arm64
# Overridden by CI with the commit SHA; defaults to "dev" for local builds.
ARG VERSION=dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Cross-compile natively — no emulation involved. -X stamps the git SHA
# into the binary (reported in startup logs and OTel service.version).
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o bin/api ./cmd/api

# ── Runtime stage ─────────────────────────────────────────────────────────────
# distroless/base:nonroot ships ca-certificates + tzdata and runs as uid 65532.
# base (not static) is required because Google ID-token verification (JWKS
# fetch) needs CA roots at runtime.
FROM gcr.io/distroless/base:nonroot

WORKDIR /app
COPY --from=builder --chown=nonroot:nonroot /app/bin/api .

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/api"]