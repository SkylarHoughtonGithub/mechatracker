# syntax=docker/dockerfile:1

# ── Build stage ───────────────────────────────────────────────────────────────
# Build context must be the repo root (not web/) so shared/ is accessible.
# Run: docker buildx build -f web/Dockerfile .
FROM --platform=$BUILDPLATFORM node:22-alpine AS builder

# NEXT_PUBLIC_GOOGLE_CLIENT_ID is baked into the client bundle at build time
# (Google Identity Services JS reads it). BACKEND_URL is server-side only —
# Next.js's /api/backend/* rewrite forwards there at request time; it is not
# bundled into the client. Both are safe defaults for local dev; CI overrides
# per-environment.
ARG NEXT_PUBLIC_GOOGLE_CLIENT_ID
ENV NEXT_PUBLIC_GOOGLE_CLIENT_ID=$NEXT_PUBLIC_GOOGLE_CLIENT_ID
ARG BACKEND_URL=http://localhost:8080
ENV BACKEND_URL=$BACKEND_URL

WORKDIR /app

# Copy workspace manifests first for layer caching
COPY package.json package-lock.json ./
COPY web/package.json ./web/
COPY shared/package.json ./shared/

# Install all workspace dependencies
RUN npm ci

# Copy source
COPY shared/ ./shared/
COPY web/ ./web/

# Build
WORKDIR /app/web
RUN npm run build

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM node:22-alpine AS runner

ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

WORKDIR /app

# Copy built output from builder
COPY --from=builder /app/web/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/web/.next/standalone/web/ ./
COPY --from=builder --chown=nextjs:nodejs /app/web/.next/standalone/node_modules ./node_modules
COPY --from=builder --chown=nextjs:nodejs /app/web/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

ENTRYPOINT ["node", "server.js"]