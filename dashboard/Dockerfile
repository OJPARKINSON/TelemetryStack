
FROM node:23-alpine AS base

ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /app

FROM base AS deps
RUN apk add --no-cache libc6-compat

COPY package.json package-lock.json* ./

RUN npm ci --ignore-scripts && npm cache clean --force

FROM base AS builder
WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules

COPY . .

ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=development
ENV ANALYZE=false

ENV NODE_OPTIONS="--max-old-space-size=1024"

RUN npm run build

FROM base AS runner
WORKDIR /app

ENV NODE_ENV=development
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public

RUN mkdir .next && chown nextjs:nodejs .next

COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./

COPY --from=builder --chown=nextjs:nodejs /app/.next ./.next
COPY --from=builder --chown=nextjs:nodejs /app/node_modules ./node_modules

USER nextjs

EXPOSE 3000

ENV HOSTNAME="0.0.0.0"
ENV PORT=3000

HEALTHCHECK --interval=60s --timeout=30s --start-period=80s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3000/api/health', (res) => { process.exit(res.statusCode === 200 ? 0 : 1) })" || exit 1

ENV NODE_OPTIONS="--max-old-space-size=512"

CMD ["sh", "-c", "if [ -f server.js ]; then node server.js; else npm start; fi"]