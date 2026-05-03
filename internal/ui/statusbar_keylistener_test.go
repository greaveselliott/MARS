package ui

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type fakeStatusProvider struct {
	paused  bool
	healthy bool
}

func (p *fakeStatusProvider) IsPaused() bool { return p.paused }
func (p *fakeStatusProvider) Healthy() bool  { return p.healthy }

func TestStatusBarRedrawStates(t *testing.T) {
	cases := []struct {
		name    string
		paused  bool
		healthy bool
		want    string
	}{
		{name: "running", healthy: true, want: "RUNNING"},
		{name: "paused", paused: true, healthy: true, want: "PAUSED"},
		{name: "stopped", healthy: false, want: "STOPPED"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			sb := NewStatusBar(&buf, &fakeStatusProvider{paused: tc.paused, healthy: tc.healthy})
			sb.Redraw()
			if !bytes.Contains(buf.Bytes(), []byte(tc.want)) {
				t.Fatalf("status bar output %q missing %q", buf.String(), tc.want)
			}
		})
	}
}

func TestStatusBarFlashAndPrintAbove(t *testing.T) {
	var buf bytes.Buffer
	sb := NewStatusBar(&buf, &fakeStatusProvider{healthy: true})

	sb.Flash("Scan complete")
	if !bytes.Contains(buf.Bytes(), []byte("Scan complete")) {
		t.Fatalf("flash output %q missing message", buf.String())
	}

	sb.PrintAbove("hello operator")
	if !bytes.Contains(buf.Bytes(), []byte("hello operator")) {
		t.Fatalf("print-above output %q missing message", buf.String())
	}
}

type fakeController struct {
	paused   bool
	pause    int
	resume   int
	restart  int
	scan     int
	failNext error
}

func (c *fakeController) Pause() { c.paused = true; c.pause++ }
func (c *fakeController) Resume() {
	c.paused = false
	c.resume++
}
func (c *fakeController) IsPaused() bool { return c.paused }
func (c *fakeController) Restart(context.Context) error {
	c.restart++
	return c.failNext
}
func (c *fakeController) ScanAllRepos(context.Context) error {
	c.scan++
	return c.failNext
}

func TestKeyListenerDispatchesControls(t *testing.T) {
	var buf bytes.Buffer
	ctrl := &fakeController{}
	stopped := false
	sb := NewStatusBar(&buf, &fakeStatusProvider{healthy: true})
	kl := NewKeyListener(ctrl, func() { stopped = true }, sb)

	kl.dispatch(context.Background(), 'p')
	if ctrl.pause != 1 || !ctrl.paused {
		t.Fatalf("pause not dispatched: %+v", ctrl)
	}
	kl.dispatch(context.Background(), 'p')
	if ctrl.resume != 1 || ctrl.paused {
		t.Fatalf("resume not dispatched: %+v", ctrl)
	}
	kl.dispatch(context.Background(), 'r')
	if ctrl.restart != 1 {
		t.Fatalf("restart not dispatched: %+v", ctrl)
	}
	kl.dispatch(context.Background(), 's')
	if ctrl.scan != 1 {
		t.Fatalf("scan not dispatched: %+v", ctrl)
	}
	kl.dispatch(context.Background(), 'q')
	if !stopped {
		t.Fatal("quit did not call stop function")
	}
}

func TestKeyListenerDispatchReportsActionFailures(t *testing.T) {
	var buf bytes.Buffer
	ctrl := &fakeController{failNext: errors.New("boom")}
	sb := NewStatusBar(&buf, &fakeStatusProvider{healthy: true})
	kl := NewKeyListener(ctrl, nil, sb)

	kl.dispatch(context.Background(), 'r')
	kl.dispatch(context.Background(), 's')

	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("Restart failed: boom")) {
		t.Fatalf("restart failure missing from %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("Scan failed: boom")) {
		t.Fatalf("scan failure missing from %q", out)
	}
}
