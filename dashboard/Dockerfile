FROM node:23-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ENV VITE_BASE_PATH=/dash/
RUN npm run build

FROM caddy:alpine
COPY --from=build /app/dist /app/www
COPY Caddyfile /etc/caddy/Caddyfile
EXPOSE 8080
