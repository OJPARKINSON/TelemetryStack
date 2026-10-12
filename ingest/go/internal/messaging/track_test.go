package messaging

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/OJPARKINSON/ibt/headers"
)

func trackSession(trackID int, pcts ...float64) *headers.Session {
	sess := &headers.Session{}
	sess.WeekendInfo.TrackID = trackID
	sess.WeekendInfo.TrackDisplayName = "Daytona Road"
	for i, p := range pcts {
		sess.SplitTimeInfo.Sectors = append(sess.SplitTimeInfo.Sectors, headers.Sectors{SectorNum: i, SectorStartPct: p})
	}
	return sess
}

func TestPutTrack(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody trackBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL + "/api/ingest")
	if err := PutTrack(context.Background(), srv.Client(), cfg, trackSession(192, 0, 0.25, 0.5)); err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPut || gotPath != "/api/tracks/192" {
		t.Fatalf("got %s %s, want PUT /api/tracks/192", gotMethod, gotPath)
	}
	want := trackBody{TrackName: "Daytona Road", SectorStartPct: []float64{0, 0.25, 0.5}}
	if !reflect.DeepEqual(gotBody, want) {
		t.Fatalf("body %+v, want %+v", gotBody, want)
	}
}

func TestPutTrackSkips(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL + "/api/ingest")
	for name, sess := range map[string]*headers.Session{
		"nil session": nil,
		"no track id": trackSession(0, 0, 0.5),
		"no sectors":  trackSession(192),
	} {
		if err := PutTrack(context.Background(), srv.Client(), cfg, sess); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestPutTrackServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	if err := PutTrack(context.Background(), srv.Client(), testConfig(srv.URL+"/api/ingest"), trackSession(192, 0)); err == nil {
		t.Fatal("want error on 400")
	}
}
