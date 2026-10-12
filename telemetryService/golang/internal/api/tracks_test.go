package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/ojparkinson/telemetryService/internal/domain"
)

func serveTracks(f *fakeSessions, method, path, body string) *httptest.ResponseRecorder {
	s := NewServer("", f, &fakeWriter{})
	s.logger = log.New(io.Discard, "", 0)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.app.ServeHTTP(rec, req)
	return rec
}

func TestPutTrack(t *testing.T) {
	cases := []struct {
		name   string
		id     string
		body   string
		status int
	}{
		{"valid", "192", `{"track_name":"Daytona Road","sector_start_pct":[0,0.137196,0.273519,0.353413,0.630555]}`, http.StatusNoContent},
		{"single sector", "192", `{"track_name":"x","sector_start_pct":[0]}`, http.StatusNoContent},
		{"non-digit id", "abc", `{"sector_start_pct":[0]}`, http.StatusBadRequest},
		{"bad json", "192", `{`, http.StatusBadRequest},
		{"missing sectors", "192", `{"track_name":"x"}`, http.StatusBadRequest},
		{"first not zero", "192", `{"sector_start_pct":[0.1,0.5]}`, http.StatusBadRequest},
		{"not ascending", "192", `{"sector_start_pct":[0,0.5,0.5]}`, http.StatusBadRequest},
		{"descending", "192", `{"sector_start_pct":[0,0.5,0.3]}`, http.StatusBadRequest},
		{"one or more", "192", `{"sector_start_pct":[0,1]}`, http.StatusBadRequest},
		{"negative", "192", `{"sector_start_pct":[0,-0.1]}`, http.StatusBadRequest},
		{"too many", "192", `{"sector_start_pct":[` + manySectors(65) + `]}`, http.StatusBadRequest},
		{"max allowed", "192", `{"sector_start_pct":[` + manySectors(64) + `]}`, http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeSessions{}
			rec := serveTracks(f, http.MethodPut, "/api/tracks/"+tc.id, tc.body)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tc.status, rec.Body)
			}
			if _, stored := f.tracks[tc.id]; stored != (tc.status == http.StatusNoContent) {
				t.Fatalf("stored = %v, want %v", stored, !stored)
			}
		})
	}
}

func manySectors(n int) string {
	parts := make([]string, n)
	for i := range parts {
		b, _ := json.Marshal(float64(i) / float64(n))
		parts[i] = string(b)
	}
	return strings.Join(parts, ",")
}

func TestPutTrackStoresBody(t *testing.T) {
	f := &fakeSessions{}
	serveTracks(f, http.MethodPut, "/api/tracks/192", `{"track_name":"Daytona Road","sector_start_pct":[0,0.5]}`)

	want := domain.Track{TrackID: "192", TrackName: "Daytona Road", SectorStartPct: []float64{0, 0.5}}
	if got := f.tracks["192"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("stored %+v, want %+v", got, want)
	}
}

func TestGetLapTrack(t *testing.T) {
	track := domain.Track{TrackID: "192", TrackName: "Daytona Road", SectorStartPct: []float64{0, 0.5}}
	f := &fakeSessions{
		tracks:   map[string]domain.Track{"192": track},
		lapTrack: map[string]string{"7/3": "192", "7/4": "999"},
	}

	cases := []struct {
		name   string
		path   string
		status int
	}{
		{"found", "/api/sessions/7/laps/3/track", http.StatusOK},
		{"unknown lap", "/api/sessions/7/laps/9/track", http.StatusNotFound},
		{"track never sent", "/api/sessions/7/laps/4/track", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveTracks(f, http.MethodGet, tc.path, "")
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			if tc.status != http.StatusOK {
				return
			}
			var got domain.Track
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, track) {
				t.Fatalf("got %+v, want %+v", got, track)
			}
		})
	}
}
