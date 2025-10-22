module github.com/OJPARKINSON/IRacing-Display/ingest/go

go 1.24

require (
	github.com/OJPARKINSON/ibt v0.1.4
	github.com/fatih/color v1.18.0
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/teamjorge/ibt v0.0.0-20240923192211-5f50fa19d38d
	go.uber.org/zap v1.27.0
	golang.org/x/term v0.34.0
	google.golang.org/protobuf v1.33.0
)

// Use local fork instead of remote dependency
// replace github.com/OJPARKINSON/ibt => ./ibt

require (
	github.com/kr/pretty v0.3.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/exp v0.0.0-20240604190554-fc45aab8b7f8 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
