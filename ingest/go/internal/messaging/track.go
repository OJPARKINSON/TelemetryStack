package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/ibt/headers"
)

type trackBody struct {
	TrackName      string    `json:"track_name"`
	SectorStartPct []float64 `json:"sector_start_pct"`
}

// PutTrack sends the session's track ID, name and sector boundaries to the
// service. It is a no-op in dry runs and when the session has no track ID or no sectors.
func PutTrack(ctx context.Context, client *http.Client, cfg *config.Config, sess *headers.Session) error {
	if cfg.DryRun || sess == nil || sess.WeekendInfo.TrackID == 0 || len(sess.SplitTimeInfo.Sectors) == 0 {
		return nil
	}

	body := trackBody{TrackName: sess.WeekendInfo.TrackDisplayName}
	for _, s := range sess.SplitTimeInfo.Sectors {
		body.SectorStartPct = append(body.SectorStartPct, s.SectorStartPct)
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := strings.TrimSuffix(cfg.IngestURL, "ingest") + "tracks/" + strconv.Itoa(sess.WeekendInfo.TrackID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("put track: %w", err)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
	res.Body.Close()

	if res.StatusCode/100 != 2 {
		return fmt.Errorf("put track: server returned %d", res.StatusCode)
	}
	return nil
}
