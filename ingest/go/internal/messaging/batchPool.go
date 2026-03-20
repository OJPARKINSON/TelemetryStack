package messaging

import (
	"sync"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type BatchPool struct {
	pool sync.Pool
}

func NewBatchPool(batchSize int) *BatchPool {
	return &BatchPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &TelemetryBatch{
					Records: make([]*Telemetry, 0, batchSize),
				}
			},
		},
	}
}

func (bp *BatchPool) Get() *TelemetryBatch {
	return bp.pool.Get().(*TelemetryBatch)
}

func (bp *BatchPool) Put(batch *TelemetryBatch) {
	batch.Records = batch.Records[:0]
	batch.BatchId = ""
	batch.Timestamp = &timestamppb.Timestamp{}
	batch.WorkerId = 0
	batch.SessionId = ""
	bp.pool.Put(batch)
}
