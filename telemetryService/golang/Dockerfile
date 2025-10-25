FROM golang:1.25.1 as builder

WORKDIR /app
COPY go.mod  .
COPY go.sum  .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /telemetryService -ldflags="-w -s"

FROM gcr.io/distroless/static-debian12 as run

COPY --from=builder /telemetryService /telemetryService

WORKDIR /app

CMD ["/telemetryService"]