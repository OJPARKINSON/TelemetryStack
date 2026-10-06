FROM golang:1.26-alpine AS builder

ARG TARGETARCH

WORKDIR /build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
  -o telemetry-service \
  -trimpath \
  -ldflags="-w -s" \
  ./cmd/telemetry-service

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /build/telemetry-service /telemetry-service

USER nonroot:nonroot

EXPOSE 8010

ENTRYPOINT ["/telemetry-service"]