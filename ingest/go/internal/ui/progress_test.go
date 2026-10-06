package ui

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func step(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func TestModelLifecycle(t *testing.T) {
	files := []File{{"a.ibt", 48_200_000}, {"b.ibt", 1_000}, {"c.ibt", 10}, {"d.ibt", 1}, {"e.ibt", 1}, {"f.ibt", 1}, {"g.ibt", 1}, {"h.ibt", 1}}
	m := NewModel(files, nil)

	// Two groups in one file: progress must accumulate, not reset.
	m = step(m, startMsg{"a.ibt", 100, 2*time.Hour + 54*time.Minute + 30*time.Second}, batchMsg{"a.ibt", 30}, batchMsg{"a.ibt", 33})
	v := m.View()
	for _, want := range []string{"63%", "48.2 MB", "Up next", "… 2 more"} {
		if !strings.Contains(v, want) {
			t.Errorf("active view missing %q:\n%s", want, v)
		}
	}

	m = step(m, completeMsg{"a.ibt"}, startMsg{"b.ibt", 10, time.Hour}, failedMsg{"b.ibt", errors.New("boom")}, doneMsg{elapsed: 3 * time.Second})
	v = m.View()
	for _, want := range []string{"✓", "✗", "boom", "Done: 1 files uploaded in 3s · 2 hours and 54 minutes of racing", "1 failed"} {
		if !strings.Contains(v, want) {
			t.Errorf("final view missing %q:\n%s", want, v)
		}
	}
	if strings.Index(v, "a.ibt") > strings.Index(v, "b.ibt") {
		t.Errorf("finished rows not in completion order:\n%s", v)
	}
}

func TestFinishedRowsCapped(t *testing.T) {
	var files []File
	for i := range finishedLimit + 3 {
		files = append(files, File{Name: fmt.Sprintf("f%02d.ibt", i)})
	}
	m := NewModel(files, nil)
	for _, f := range files {
		m = step(m, startMsg{f.Name, 1, 0}, completeMsg{f.Name})
	}
	v := m.View()
	if strings.Count(v, "✓") != finishedLimit || !strings.Contains(v, "… 3 earlier") || strings.Contains(v, "f00.ibt") {
		t.Errorf("finished rows not capped to the most recent %d:\n%s", finishedLimit, v)
	}
}

func TestPlainReporter(t *testing.T) {
	var buf bytes.Buffer
	r := NewPlainReporter(&buf, []File{{"a.ibt", 2_000}})
	r.OnFileStart("a.ibt", 10, 30*time.Second)
	r.OnBatchSent("a.ibt", 10)
	r.OnFileComplete("a.ibt")
	r.Done(time.Second, false)
	r.OnFileComplete("a.ibt") // a stray event after Done must not reprint the summary

	out := buf.String()
	if strings.Count(out, "\n") != 2 || !strings.Contains(out, "✓") || !strings.Contains(out, "Done: 1 files") {
		t.Errorf("unexpected plain output:\n%s", out)
	}
}

func TestHumanDuration(t *testing.T) {
	tests := map[time.Duration]string{
		45 * time.Minute:        "45 minutes",
		time.Hour:               "1 hour",
		time.Hour + time.Minute: "1 hour and 1 minute",
		2*time.Hour + 54*time.Minute + 59*time.Second: "2 hours and 54 minutes",
		16*time.Hour + 54*time.Minute:                 "16 hours and 54 minutes",
	}
	for d, want := range tests {
		if got := humanDuration(d); got != want {
			t.Errorf("humanDuration(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestShadeBar(t *testing.T) {
	tests := []struct {
		pct        float64
		want, deny string
	}{
		{0, "  0%", "█▓▒░"},
		{1.0 / barWidth, "▓", "█"},
		{0.5, strings.Repeat("█", 9) + "▓▒░", ""},
		{1, strings.Repeat("█", barWidth) + " 100%", "░"},
	}
	for _, tt := range tests {
		got := shadeBar(tt.pct, barWidth)
		if !strings.Contains(got, tt.want) {
			t.Errorf("shadeBar(%v) = %q, want it to contain %q", tt.pct, got, tt.want)
		}
		if tt.deny != "" && strings.ContainsAny(got, tt.deny) {
			t.Errorf("shadeBar(%v) = %q, must not contain any of %q", tt.pct, got, tt.deny)
		}
	}
}
