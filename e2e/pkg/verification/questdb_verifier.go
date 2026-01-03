package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"time"
)

type response struct {
	Query     string          `json:"query"`
	Columns   []column        `json:"columns"` // Should be array of column objects
	Timestamp int64           `json:"timestamp"`
	Dataset   [][]interface{} `json:"dataset"` // Mixed types in QuestDB responses
	Count     int             `json:"count"`
}

type column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func GetRecordCount() (int, error) {
	u, err := url.Parse("http://localhost:9000")
	if err != nil {
		return -1, fmt.Errorf("error parsing url, ", err)
	}

	u.Path += "exec"
	params := url.Values{}
	params.Add("query", `
		SELECT count(timestamp) FROM TelemetryTicks
	`)
	u.RawQuery = params.Encode()
	url := fmt.Sprintf("%v", u)

	res, err := http.Get(url)
	if err != nil {
		return -1, fmt.Errorf("error to get stored records, ", err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("error: failed ro read body, ", err)
	}

	response := response{}

	if err := json.Unmarshal(body, &response); err != nil {
		return -1, fmt.Errorf("failed to unmarshal: %w, body: %s", err, string(body))
	}

	if len(response.Dataset) == 0 || len(response.Dataset[0]) == 0 {
		log.Printf("empty dataset in response")
	}

	actualCount := int(response.Dataset[0][0].(float64)) // JSON numbers are float64

	log.Printf("✓ Count validation passed: %d records", actualCount)

	return actualCount, nil
}

func waitForRecordCountWithMetrics(ctx context.Context, expectedCount int, timeout time.Duration) (*ThroughputMetrics, error) {
	metrics := &ThroughputMetrics{
		StartTime: time.Now(),
		Samples:   []Sample{},
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastCount := 0
	lastTime := time.Now()

	for {
		now := time.Now()
		count, err := GetRecordCount()
		if err != nil {
			return nil, err
		}

		// Calculate instantaneous throughput
		deltaRecords := count - lastCount
		deltaTime := now.Sub(lastTime).Seconds()
		instantThroughput := float64(deltaRecords) / deltaTime

		// Record sample
		metrics.Samples = append(metrics.Samples, Sample{
			Timestamp:  now,
			Count:      count,
			Throughput: instantThroughput,
		})

		progress := float64(count) / float64(expectedCount) * 100
		log.Printf("Progress: %d/%d (%.1f%%) | Throughput: %.0f rec/sec",
			count, expectedCount, progress, instantThroughput)

		if count >= expectedCount {
			metrics.EndTime = now
			metrics.TotalRecords = count
			return metrics, nil
		}

		if now.After(deadline) {
			return metrics, fmt.Errorf("timeout: %d/%d records after %v",
				count, expectedCount, timeout)
		}

		lastCount = count
		lastTime = now

		<-ticker.C
	}
}

type ThroughputMetrics struct {
	StartTime    time.Time
	EndTime      time.Time
	TotalRecords int
	Samples      []Sample
}

type Sample struct {
	Timestamp  time.Time
	Count      int
	Throughput float64
}

func (m *ThroughputMetrics) AvgThroughput() float64 {
	if m.EndTime.IsZero() {
		return 0
	}
	return float64(m.TotalRecords) / m.EndTime.Sub(m.StartTime).Seconds()
}

func (m *ThroughputMetrics) PeakThroughput() float64 {
	peak := 0.0
	for _, s := range m.Samples {
		if s.Throughput > peak {
			peak = s.Throughput
		}
	}
	return peak
}

func (m *ThroughputMetrics) P95Throughput() float64 {
	if len(m.Samples) == 0 {
		return 0
	}

	throughputs := make([]float64, len(m.Samples))
	for i, s := range m.Samples {
		throughputs[i] = s.Throughput
	}
	sort.Float64s(throughputs)

	idx := int(float64(len(throughputs)) * 0.95)
	return throughputs[idx]
}
