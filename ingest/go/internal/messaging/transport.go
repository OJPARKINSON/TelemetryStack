package messaging

import (
	"net/http"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
)

// NewHTTPClient builds the client every PubSub shares. http.DefaultTransport keeps only
// 2 idle connections per host, so WORKER_COUNT publishers would redial for most batches.
func NewHTTPClient(cfg *config.Config) *http.Client {
	workers := cfg.WorkerCount
	if workers < 1 {
		workers = 1
	}

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConnsPerHost = workers
	t.MaxIdleConns = workers * 2
	t.MaxConnsPerHost = workers * 2
	t.WriteBufferSize = 256 << 10 // bodies are megabytes, not kilobytes

	return &http.Client{
		Transport: t,
		Timeout:   30 * time.Second,
	}
}
