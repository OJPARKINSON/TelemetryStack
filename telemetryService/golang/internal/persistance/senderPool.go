package persistance

import (
	"context"
	"fmt"
	"time"

	qdb "github.com/questdb/go-questdb-client/v4"
)

type SenderPool struct {
	pool chan qdb.LineSender
	size int
	host string
	port int
}

func NewSenderPool(size int, host string, port int) (*SenderPool, error) {
	pool := &SenderPool{
		pool: make(chan qdb.LineSender, size),
		size: size,
		host: host,
		port: port,
	}

	for i := 0; i < size; i++ {
		sender, err := qdb.NewLineSender(
			context.Background(),
			qdb.WithHttp(),
			qdb.WithAddress(fmt.Sprintf("%s:%d", host, port)),
			qdb.WithAutoFlushRows(10000),
			qdb.WithRequestTimeout(60*time.Second),
		)

		if err != nil {
			return nil, fmt.Errorf("failed to create sender, %d: %w", i, err)
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
