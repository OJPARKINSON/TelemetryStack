package persistance

import (
	"context"
	"fmt"
	"time"

	"github.com/ojparkinson/telemetryService/internal/config"
	qdb "github.com/questdb/go-questdb-client/v4"
)

type SenderPool struct {
	pool chan qdb.LineSender
	size int
	host string
	port int
}

func NewSenderPool(config *config.Config) (*SenderPool, error) {
	pool := &SenderPool{
		pool: make(chan qdb.LineSender, config.QuestPoolSize),
		size: config.QuestPoolSize,
		host: config.QuestDbHost,
		port: config.QuestDBPort,
	}

	maxRetries := 10
	baseDelay := 1 * time.Second

	for i := 0; i < config.QuestPoolSize; i++ {
		var sender qdb.LineSender
		var err error

		// Retry sender creation with exponential backoff
		for attempt := 0; attempt < maxRetries; attempt++ {
			sender, err = qdb.NewLineSender(
				context.Background(),
				qdb.WithHttp(),
				qdb.WithAddress(fmt.Sprintf("%s:%d", config.QuestDbHost, config.QuestDBPort)),
				qdb.WithAutoFlushRows(10000),
				qdb.WithRequestTimeout(60*time.Second),
			)

			if err == nil {
				break
			}

			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<uint(attempt))
				fmt.Printf("QuestDB sender creation failed for pool #%d (attempt %d/%d), retrying in %v: %v\n", i, attempt+1, maxRetries, delay, err)
				time.Sleep(delay)
			} else {
				return nil, fmt.Errorf("failed to create sender %d after %d retries: %w", i, maxRetries, err)
			}
		}

		pool.pool <- sender
	}

	return pool, nil
}

func (p *SenderPool) Get() qdb.LineSender {
	return <-p.pool
}

func (p *SenderPool) Return(sender qdb.LineSender) {
	p.pool <- sender
}

func (p *SenderPool) Close() {
	close(p.pool)
	for sender := range p.pool {
		sender.Close(context.Background())
	}
}
