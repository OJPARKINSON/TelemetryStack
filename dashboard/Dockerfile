FROM ghcr.io/pnpm/pnpm:12.9.1 AS web
WORKDIR /app
ENV VITE_BASE_PATH=/dashboard/ npm_config_store_dir=/pnpm/store
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN --mount=type=cache,target=/pnpm/store pnpm fetch
COPY . .
RUN --mount=type=cache,target=/pnpm/store pnpm ci --offline && pnpm build

FROM caddy:2.11 AS caddy

FROM gcr.io/distroless/static:nonroot
COPY --from=caddy /usr/bin/caddy /caddy
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=web /app/dist /srv
ENV XDG_CONFIG_HOME=/tmp XDG_DATA_HOME=/tmp
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
