package ui

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/sys/unix"
)

// Controller is the set of operations the key listener can trigger.
// Implemented by *serve.Server.
type Controller interface {
	Pause()
	Resume()
	IsPaused() bool
	Restart(ctx context.Context) error
	ScanAllRepos(ctx context.Context) error
}

// KeyListener reads single keystrokes from stdin while the terminal
// is in raw mode and dispatches to the Controller.
type KeyListener struct {
	ctrl      Controller
	cancel    context.CancelFunc
	stopFunc  func() // called when the user presses 'q'
	statusBar *StatusBar
}

// NewKeyListener creates a key listener that dispatches to ctrl.
// stopFunc is called when the user presses 'q' to request a graceful shutdown.
// An optional StatusBar is updated after each action.
func NewKeyListener(ctrl Controller, stopFunc func(), sb *StatusBar) *KeyListener {
	return &KeyListener{ctrl: ctrl, stopFunc: stopFunc, statusBar: sb}
}

// Start begins reading keystrokes in a goroutine. It sets the terminal
// to raw mode and restores the original state when the context is done
// or Stop is called.
func (kl *KeyListener) Start(ctx context.Context) {
	ctx, kl.cancel = context.WithCancel(ctx)

	fd := int(os.Stdin.Fd())
	origTermios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		slog.Warn("ui: cannot set raw mode — interactive controls disabled", "err", err)
		return
	}

	raw := *origTermios
	raw.Lflag &^= unix.ICANON | unix.ECHO
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &raw); err != nil {
		slog.Warn("ui: cannot set raw mode — interactive controls disabled", "err", err)
		return
	}

	go func() {
		defer func() {
			_ = unix.IoctlSetTermios(fd, unix.TIOCSETA, origTermios)
		}()

		buf := make([]byte, 1)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, readErr := os.Stdin.Read(buf)
			if readErr != nil || n == 0 {
				return
			}

			kl.dispatch(ctx, buf[0])
		}
	}()
}

// Stop terminates the key listener goroutine.
func (kl *KeyListener) Stop() {
	if kl.cancel != nil {
		kl.cancel()
	}
}

func (kl *KeyListener) dispatch(ctx context.Context, key byte) {
	switch key {
	case 'p', 'P':
		if kl.ctrl.IsPaused() {
			kl.ctrl.Resume()
			kl.notify("Resumed")
		} else {
			kl.ctrl.Pause()
			kl.notify("Paused — running jobs will complete, no new jobs claimed")
		}

	case 'r', 'R':
		kl.notify("Restarting…")
		if err := kl.ctrl.Restart(ctx); err != nil {
			kl.notify(fmt.Sprintf("Restart failed: %v", err))
		} else {
			kl.notify("Restart complete")
		}

	case 's', 'S':
		kl.notify("Scanning repos…")
		if err := kl.ctrl.ScanAllRepos(ctx); err != nil {
			kl.notify(fmt.Sprintf("Scan failed: %v", err))
		} else {
			kl.notify("Scan complete")
		}

	case 'q', 'Q':
		kl.notify("Shutting down…")
		if kl.stopFunc != nil {
			kl.stopFunc()
		}

	case 'h', 'H', '?':
		kl.printHelp()
	}

	if kl.statusBar != nil {
		kl.statusBar.Redraw()
	}
}

func (kl *KeyListener) notify(msg string) {
	if kl.statusBar != nil {
		kl.statusBar.Flash(msg)
	} else {
		fmt.Fprintf(os.Stderr, "\r\033[K  → %s\n", msg)
	}
}

func (kl *KeyListener) printHelp() {
	help := "\r\033[K" +
		"  \033[1mKey bindings:\033[0m\n" +
		"  \033[1mp\033[0m  Pause / Resume\n" +
		"  \033[1mr\033[0m  Warm restart\n" +
		"  \033[1ms\033[0m  Re-scan repos\n" +
		"  \033[1mq\033[0m  Graceful stop\n" +
		"  \033[1mh\033[0m  Show this help\n"
	fmt.Fprint(os.Stderr, help)
}
