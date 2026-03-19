package questdb

import (
	"context"

	"github.com/ojparkinson/telemetryService/internal/messaging"
)

func (r *Repository) WriteBatch(ctx context.Context, records []*messaging.Telemetry) error {
	sender := r.writers.Get()
	defer r.writers.Return(sender)

	return nil
}
