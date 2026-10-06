package worker

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"go.uber.org/zap"
)

type failRecorder struct {
	failed []string
	pool   *WorkerPool
	// countAtReport is TotalFilesFailed when OnFileFailed ran.
	countAtReport int
}

func (f *failRecorder) OnFileStart(string, int, time.Duration) {}
func (f *failRecorder) OnBatchSent(string, int)                {}
func (f *failRecorder) OnFileComplete(string)                  {}
func (f *failRecorder) OnFileFailed(name string, _ error) {
	f.failed = append(f.failed, name)
	f.countAtReport = f.pool.GetMetrics().TotalFilesFailed
}

// A retryable file that disappears before its retry must still count as failed,
// otherwise waitForCompletion's processed+failed >= expected never holds.
func TestHandleErrorVanishedFileCountsAsFailed(t *testing.T) {
	rec := &failRecorder{}
	pool := NewWorkerPool(&config.Config{WorkerCount: 1, FileQueueSize: 1, MaxRetries: 3}, zap.NewNop(), rec)
	rec.pool = pool

	pool.handleError(WorkError{
		FilePath: filepath.Join(t.TempDir(), "gone.ibt"),
		Error:    errors.New("boom"),
		Retry:    true,
	})

	if got := pool.GetMetrics().TotalFilesFailed; got != 1 {
		t.Fatalf("TotalFilesFailed = %d, want 1", got)
	}
	if len(rec.failed) != 1 || rec.failed[0] != "gone.ibt" {
		t.Fatalf("OnFileFailed calls = %v, want [gone.ibt]", rec.failed)
	}
	if rec.countAtReport != 0 {
		t.Fatalf("failure was counted before it was reported; waitForCompletion could exit and drop the row")
	}
}
