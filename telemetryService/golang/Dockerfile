FROM golang:1.25-bookworm as base

WORKDIR /app
COPY go.mod  .
COPY go.sum  .

RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -o /telemetryService -ldflags="-w -s"

CMD ["/telemetryService"]
