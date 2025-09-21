FROM node:22-alpine AS base

ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /app

# Install dependencies only when needed
FROM base AS deps
RUN apk add --no-cache libc6-compat

# Copy package files
COPY package.json package-lock.json* ./

# Use BuildKit cache mount for npm cache (works with any driver)
RUN --mount=type=cache,target=/root/.npm \
  npm ci --only=production --ignore-scripts

# Install dev dependencies in separate layer for build
FROM base AS dev-deps
RUN apk add --no-cache libc6-compat

COPY package.json package-lock.json* ./

RUN --mount=type=cache,target=/root/.npm \
  npm ci --ignore-scripts

# Build the application
FROM base AS builder
WORKDIR /app

# Copy all dependencies
COPY --from=dev-deps /app/node_modules ./node_modules
COPY . .

# Build environment variables
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production
ENV NODE_OPTIONS="--max-old-space-size=4096"

# Use BuildKit cache mount for Next.js cache (internal cache only)
RUN --mount=type=cache,target=/app/.next/cache \
  --mount=type=cache,target=/root/.npm \
  npm run build

# Production image
FROM base AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs && \
  adduser --system --uid 1001 nextjs

# Copy production dependencies
COPY --from=deps /app/node_modules ./node_modules
COPY --from=builder /app/public ./public

# Copy Next.js build output
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV HOSTNAME="0.0.0.0"
ENV PORT=3000

# Simplified healthcheck
HEALTHCHECK --interval=60s --timeout=30s --start-period=30s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/api/health || exit 1

CMD ["node", "server.js"]