module github.com/OJPARKINSON/IRacing-Display/ingest/go

go 1.26.1

require (
	github.com/OJPARKINSON/ibt v0.1.5
	github.com/prometheus/client_golang v1.23.2
	go.uber.org/zap v1.27.1
	golang.org/x/sync v0.19.0
	google.golang.org/protobuf v1.36.11
)

require github.com/joho/godotenv v1.5.1 // indirect

// // Use local fork instead of remote dependency
// replace github.com/OJPARKINSON/ibt => ./ibt

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
