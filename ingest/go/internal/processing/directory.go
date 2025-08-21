package processing

import (
	"os"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"go.uber.org/zap"
)

type Directory struct {
	path             string
	lastScan         time.Time
	fileAgeThreshold time.Duration
	logger           *zap.Logger
}

func NewDir(path string, cfg *config.Config, logger *zap.Logger) *Directory {
	return &Directory{
		path:             path,
		fileAgeThreshold: cfg.FileAgeThreshold,
		logger:           logger,
	}
}

func (d *Directory) WatchDir() []os.DirEntry {
	d.lastScan = time.Now()
	files, err := os.ReadDir(d.path)
	if err != nil {
		d.logger.Fatal("Could not read the given input directory", zap.Error(err))
	}

	filesToProcess := make([]os.DirEntry, 0)

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			d.logger.Warn("Could not get file info", zap.String("file", file.Name()), zap.Error(err))
			continue
		}

		if info.ModTime().Before(time.Now().Add(-d.fileAgeThreshold)) {
			filesToProcess = append(filesToProcess, file)
		} else {
			d.logger.Debug("Skipping recent file (still being written?)",
				zap.Duration("age_threshold", d.fileAgeThreshold),
				zap.String("file", file.Name()))
		}
	}

	d.lastScan = time.Now()
	d.logger.Info("Files found for processing",
		zap.Int("ready_files", len(filesToProcess)),
		zap.Int("total_files", len(files)))
	return filesToProcess
}
