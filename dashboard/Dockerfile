FROM node:23-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ENV VITE_BASE_PATH=/dashboard/
RUN npm run build

FROM devforth/spa-to-http:latest
COPY --from=build /app/dist/ .
CMD ["--base-path", "/dashboard", "--brotli"]
