// Package ui renders ingest progress: a live bubbletea view on a TTY, plain lines otherwise.
package ui

import (
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	upNextLimit   = 5
	finishedLimit = 10 // re-rendered every spinner tick, so only the most recent are shown
	barWidth      = 24
)

// File is one queued .ibt file, in upload order.
type File struct {
	Name string
	Size int64
}

type state int

const (
	queued state = iota
	active
	done
	failed
)

type row struct {
	File
	total, sent int
	onTrack     time.Duration
	start       time.Time
	dur         time.Duration
	state       state
	err         error
}

type (
	startMsg struct {
		name    string
		total   int
		onTrack time.Duration
	}
	batchMsg struct {
		name string
		n    int
	}
	completeMsg struct{ name string }
	failedMsg   struct {
		name string
		err  error
	}
	doneMsg struct {
		elapsed     time.Duration
		interrupted bool
	}
)

var (
	green = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	red   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	faint = lipgloss.NewStyle().Faint(true)
)

// Model is the live view. cancel is called on ctrl+c.
type Model struct {
	rows     []row
	idx      map[string]int
	finished []int
	nameW    int
	spin     spinner.Model
	cancel   func()
	summary  string
}

// NewModel builds the view with every file queued, in the given order.
func NewModel(files []File, cancel func()) Model {
	m := Model{
		idx:    make(map[string]int, len(files)),
		spin:   spinner.New(spinner.WithSpinner(spinner.Dot)),
		cancel: cancel,
	}
	for i, f := range files {
		m.rows = append(m.rows, row{File: f})
		m.idx[f.Name] = i
		m.nameW = max(m.nameW, len(f.Name))
	}
	return m
}

func (m Model) Init() tea.Cmd { return m.spin.Tick }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" && m.cancel != nil {
			m.cancel()
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case startMsg:
		if r := m.row(msg.name); r != nil {
			// A retry restarts the file from zero.
			r.state, r.total, r.sent, r.onTrack, r.start = active, msg.total, 0, msg.onTrack, time.Now()
		}
	case batchMsg:
		if r := m.row(msg.name); r != nil {
			r.sent += msg.n
		}
	case completeMsg:
		m.finish(msg.name, done, nil)
	case failedMsg:
		m.finish(msg.name, failed, msg.err)
	case doneMsg:
		verb := "Done"
		if msg.interrupted {
			verb = "Stopped"
		}
		m.summary = fmt.Sprintf("%s: %d files uploaded in %s", verb, m.count(done), msg.elapsed.Round(time.Millisecond))
		if racing := m.onTrack(); racing >= time.Minute {
			m.summary += " · " + humanDuration(racing) + " of racing"
		}
		if n := m.count(failed); n > 0 {
			m.summary += red.Render(fmt.Sprintf(", %d failed", n))
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) row(name string) *row {
	if i, ok := m.idx[name]; ok {
		return &m.rows[i]
	}
	return nil
}

func (m *Model) finish(name string, s state, err error) {
	if r := m.row(name); r != nil && r.state != s {
		r.state, r.err, r.dur = s, err, time.Since(r.start)
		m.finished = append(m.finished, m.idx[name])
	}
}

func (m Model) count(s state) int {
	n := 0
	for _, r := range m.rows {
		if r.state == s {
			n++
		}
	}
	return n
}

// onTrack sums the time covered by uploaded files; failed files don't count.
func (m Model) onTrack() time.Duration {
	var d time.Duration
	for _, r := range m.rows {
		if r.state == done {
			d += r.onTrack
		}
	}
	return d
}

func (m Model) prefix(r row) string {
	return fmt.Sprintf("%-*s  %8s", m.nameW, r.Name, humanSize(r.Size))
}

func (m Model) finishedLine(r row) string {
	if r.state == failed {
		return red.Render("✗ ") + m.prefix(r) + "  " + red.Render(shortErr(r.err))
	}
	return green.Render("✓ ") + m.prefix(r) + "  " + faint.Render(r.dur.Round(10*time.Millisecond).String())
}

func (m Model) View() string {
	var b strings.Builder
	recent := m.finished[max(0, len(m.finished)-finishedLimit):]
	if earlier := len(m.finished) - len(recent); earlier > 0 {
		b.WriteString(faint.Render(fmt.Sprintf("  … %d earlier", earlier)) + "\n")
	}
	for _, i := range recent {
		b.WriteString(m.finishedLine(m.rows[i]) + "\n")
	}

	var activeRows, upNext []row
	for _, r := range m.rows {
		switch r.state {
		case active:
			activeRows = append(activeRows, r)
		case queued:
			upNext = append(upNext, r)
		}
	}

	if len(activeRows) > 0 {
		b.WriteString("\n")
		for _, r := range activeRows {
			pct := 0.0
			if r.total > 0 {
				pct = min(float64(r.sent)/float64(r.total), 1)
			}
			b.WriteString(m.spin.View() + m.prefix(r) + "  " + shadeBar(pct, barWidth) + "\n")
		}
	}

	if len(upNext) > 0 {
		b.WriteString("\n" + faint.Render("  Up next") + "\n")
		for _, r := range upNext[:min(len(upNext), upNextLimit)] {
			b.WriteString(faint.Render("  · "+m.prefix(r)) + "\n")
		}
		if extra := len(upNext) - upNextLimit; extra > 0 {
			b.WriteString(faint.Render(fmt.Sprintf("  · … %d more", extra)) + "\n")
		}
	}

	if m.summary != "" {
		b.WriteString("\n" + m.summary + "\n")
	}
	return b.String()
}

// shadeBar draws a solid bar whose leading edge fades out: ████▓▒░   63%
func shadeBar(pct float64, width int) string {
	n := int(math.Round(pct * float64(width)))
	fade := []rune("▓▒░")
	if n == width {
		fade = nil
	} else if n < len(fade) {
		fade = fade[:n]
	}
	cells := strings.Repeat("█", n-len(fade)) + string(fade)
	return cells + strings.Repeat(" ", width-n) + fmt.Sprintf(" %3.0f%%", pct*100)
}

// shortErr keeps the innermost cause on one line; -v has the full error.
func shortErr(err error) string {
	msg, _, _ := strings.Cut(err.Error(), "\n")
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		msg = msg[i+2:]
	}
	return msg
}

// humanDuration spells out whole hours and minutes: "2 hours and 54 minutes", "1 hour", "45 minutes".
func humanDuration(d time.Duration) string {
	plural := func(n int, unit string) string {
		if n == 1 {
			return "1 " + unit
		}
		return fmt.Sprintf("%d %ss", n, unit)
	}
	h, m := int(d.Hours()), int(d.Minutes())%60
	switch {
	case h == 0:
		return plural(m, "minute")
	case m == 0:
		return plural(h, "hour")
	}
	return plural(h, "hour") + " and " + plural(m, "minute")
}

func humanSize(n int64) string {
	const unit = 1000
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "kMGT"[exp])
}

// Reporter implements processing.ProgressCallback by forwarding events to a sink.
type Reporter struct{ send func(tea.Msg) }

// NewTUIReporter forwards events to a running bubbletea program.
func NewTUIReporter(p *tea.Program) *Reporter { return &Reporter{send: p.Send} }

// NewPlainReporter prints one line per finished file to w (for non-TTY output).
func NewPlainReporter(w io.Writer, files []File) *Reporter {
	var mu sync.Mutex
	m := NewModel(files, nil)
	return &Reporter{send: func(msg tea.Msg) {
		mu.Lock()
		defer mu.Unlock()
		before := len(m.finished)
		next, _ := m.Update(msg)
		m = next.(Model)
		if len(m.finished) > before {
			fmt.Fprintln(w, m.finishedLine(m.rows[m.finished[len(m.finished)-1]]))
		}
		if _, ok := msg.(doneMsg); ok {
			fmt.Fprintln(w, m.summary)
		}
	}}
}

func (r *Reporter) OnFileStart(name string, total int, onTrack time.Duration) {
	r.send(startMsg{name, total, onTrack})
}
func (r *Reporter) OnBatchSent(name string, n int)      { r.send(batchMsg{name, n}) }
func (r *Reporter) OnFileComplete(name string)          { r.send(completeMsg{name}) }
func (r *Reporter) OnFileFailed(name string, err error) { r.send(failedMsg{name, err}) }

// Done sends the final summary.
func (r *Reporter) Done(elapsed time.Duration, interrupted bool) {
	r.send(doneMsg{elapsed: elapsed, interrupted: interrupted})
}
