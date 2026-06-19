// Package progress displays a live spinner on TTY stderr showing per-device status,
// and falls back to plain completion lines when output is piped.
package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

var spinFrames = [4]string{"|", "/", "*", "\\"}

// Tracker displays a live single-line status on stderr and exposes a log
// writer that clears the status line before printing so lines don't collide.
type Tracker struct {
	total   int
	mu      sync.Mutex
	done    int
	active  map[string]string // hostname -> current action
	isTTY   bool
	stop    chan struct{}
	stopped chan struct{}
}

func New(total int) *Tracker {
	t := &Tracker{
		total:   total,
		active:  make(map[string]string),
		isTTY:   term.IsTerminal(int(os.Stderr.Fd())),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go t.loop()
	return t
}

func (t *Tracker) DeviceStarted(hostname string) {
	t.mu.Lock()
	t.active[hostname] = "connecting"
	t.mu.Unlock()
}

func (t *Tracker) CommandStarted(hostname, cmd string) {
	t.mu.Lock()
	t.active[hostname] = cmd
	t.mu.Unlock()
}

func (t *Tracker) DeviceDone(hostname string) {
	t.mu.Lock()
	delete(t.active, hostname)
	t.done++
	n := t.done
	total := t.total
	t.mu.Unlock()
	if !t.isTTY {
		fmt.Fprintf(os.Stderr, "[%d/%d] %s done\n", n, total, hostname)
	}
}

// Stop ends the spinner and clears the status line.
func (t *Tracker) Stop() {
	close(t.stop)
	<-t.stopped
	if t.isTTY {
		t.clearLine()
	}
}

// LogWriter returns an io.Writer suitable for log.SetOutput that clears the
// status line before writing so log lines appear above the spinner cleanly.
func (t *Tracker) LogWriter() io.Writer {
	return &logWriter{t}
}

type logWriter struct{ tr *Tracker }

func (lw *logWriter) Write(p []byte) (int, error) {
	if lw.tr.isTTY {
		lw.tr.clearLine()
	}
	return os.Stderr.Write(p)
}

func (t *Tracker) clearLine() {
	fmt.Fprintf(os.Stderr, "\r%-80s\r", "")
}

func (t *Tracker) loop() {
	defer close(t.stopped)
	if !t.isTTY {
		return
	}
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	frame := 0
	for {
		select {
		case <-tick.C:
			t.mu.Lock()
			done := t.done
			parts := make([]string, 0, len(t.active))
			for h, action := range t.active {
				parts = append(parts, h+": "+action)
			}
			t.mu.Unlock()

			line := fmt.Sprintf("[%s] %d/%d", spinFrames[frame%4], done, t.total)
			if len(parts) > 0 {
				detail := strings.Join(parts, "  ")
				if len(detail) > 55 {
					detail = detail[:52] + "..."
				}
				line += "  " + detail
			}
			fmt.Fprintf(os.Stderr, "\r%-80s", line)
			frame++
		case <-t.stop:
			return
		}
	}
}
