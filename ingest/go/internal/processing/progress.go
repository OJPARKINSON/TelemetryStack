package processing

import "time"

// ProgressCallback provides real-time progress updates during file processing.
// Implementations must be safe for concurrent use: every worker calls them.
type ProgressCallback interface {
	// OnFileStart is called when a file begins processing; onTrack is the time the file covers
	OnFileStart(filename string, totalRecords int, onTrack time.Duration)

	// OnBatchSent is called after the server accepts a batch, with that batch's record count
	OnBatchSent(filename string, batchRecords int)

	// OnFileComplete is called when file processing finishes
	OnFileComplete(filename string)

	// OnFileFailed is called when a file has failed and will not be retried
	OnFileFailed(filename string, err error)
}

// NoOpProgressCallback is a default implementation that does nothing
type NoOpProgressCallback struct{}

func (n *NoOpProgressCallback) OnFileStart(filename string, totalRecords int, onTrack time.Duration) {
}
func (n *NoOpProgressCallback) OnBatchSent(filename string, batchRecords int) {}
func (n *NoOpProgressCallback) OnFileComplete(filename string)                {}
func (n *NoOpProgressCallback) OnFileFailed(filename string, err error)       {}
