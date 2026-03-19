package questdb

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ojparkinson/telemetryService/internal/db/generated"
)

type Repository struct {
	queries *generated.Queries
	writers *SenderPool
}

func NewRepository(pool *pgxpool.Pool, writers *SenderPool) *Repository {
	return &Repository{
		queries: generated.New(pool),
		writers: writers,
	}
}
