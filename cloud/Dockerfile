  FROM golang:1.26-alpine AS build
  WORKDIR /app
  COPY go.mod go.sum* ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server ./cmd/api

  FROM scratch
  COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
  COPY --from=build /server /server
  EXPOSE 8080
  CMD ["/server"]