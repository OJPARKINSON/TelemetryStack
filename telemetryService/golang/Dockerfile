  FROM golang:1.25-alpine AS builder

  WORKDIR /build

  COPY go.mod go.sum ./
  RUN go mod download

  COPY . .

  RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
      -o telemetry-service \
      -trimpath \
      -ldflags="-w -s" \
      ./cmd/telemetry-service

  FROM gcr.io/distroless/static:nonroot

  # Copy binary to root of distroless image
  COPY --from=builder /build/telemetry-service /telemetry-service

  USER nonroot:nonroot

  EXPOSE 8010

  ENTRYPOINT ["/telemetry-service"]