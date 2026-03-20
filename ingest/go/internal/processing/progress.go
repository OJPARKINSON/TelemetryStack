package processing

// ProgressCallback provides real-time progress updates during file processing
type ProgressCallback interface {
	// OnFileStart is called when a file begins processing
	OnFileStart(filename string, totalRecords int)

	// OnBatchSent is called after each batch is sent to RabbitMQ
	OnBatchSent(filename string, recordsSent int, batchNum int)

	// OnFileComplete is called when file processing finishes
	OnFileComplete(filename string)
}

// NoOpProgressCallback is a default implementation that does nothing
type NoOpProgressCallback struct{}

func (n *NoOpProgressCallback) OnFileStart(filename string, totalRecords int)         {}
func (n *NoOpProgressCallback) OnBatchSent(filename string, recordsSent int, batchNum int) {}
func (n *NoOpProgressCallback) OnFileComplete(filename string)                        {}
