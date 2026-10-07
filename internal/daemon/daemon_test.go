package daemon

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/purehate/herdr-git-hud/internal/model"
	"github.com/purehate/herdr-git-hud/internal/traceevent"
)

func TestFinishShowsToast(t *testing.T) {
	runner, toasts := recordingRunner(t)
	operation := &pendingOperation{
		Command: "push", Repo: t.TempDir(), Phase: "Contacting remote",
		Visible: true, Started: model.NowMillis(),
	}
	runner.finish(operation, 128)

	if len(*toasts) != 1 || !strings.HasPrefix((*toasts)[0][0], "✗ git push failed · ") {
		t.Fatalf("toasts = %q, want one push failure", *toasts)
	}
	if body := (*toasts)[0][1]; !strings.HasPrefix(body, "exit 128 · Contacting remote · ") {
		t.Fatalf("body = %q, want exit code and phase first", body)
	}
}

func TestFinishSkipsHiddenOperation(t *testing.T) {
	runner, toasts := recordingRunner(t)
	runner.finish(&pendingOperation{Command: "push", Repo: t.TempDir()}, 0)

	if len(*toasts) != 0 {
		t.Fatalf("toasts = %q, want none for a hidden operation", *toasts)
	}
}

func TestHandleReportsLastPhaseOnExit(t *testing.T) {
	runner, toasts := recordingRunner(t)
	runner.pending["root"] = &pendingOperation{
		SID: "root", Command: "commit", Repo: t.TempDir(),
		Visible: true, Started: model.NowMillis(),
	}

	runner.handle(traceevent.Event{Event: "child_start", SID: "root", Argv: []string{".git/hooks/pre-commit"}})
	runner.handle(traceevent.Event{Event: "exit", SID: "root", Code: 1})

	if len(*toasts) != 1 || !strings.HasPrefix((*toasts)[0][1], "exit 1 · Running pre-commit checks · ") {
		t.Fatalf("toasts = %q, want a failure naming the pre-commit phase", *toasts)
	}
	if len(runner.pending) != 0 {
		t.Fatalf("pending = %#v, want the finished operation removed", runner.pending)
	}
}

func TestListenKeepsReadingWhileToastIsShown(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	shown := make(chan struct{}, 1)
	runner, _ := recordingRunner(t)
	runner.notify = func(title, body string) error {
		shown <- struct{}{}
		<-release
		return nil
	}
	runner.pending["operation:op1"] = &pendingOperation{
		SID: "operation:op1", Label: "odoo-test", Repo: t.TempDir(),
		Visible: true, Started: model.NowMillis(),
	}
	connection, socket := listenOnTempSocket(t)
	done := make(chan error, 1)
	go func() { done <- runner.listen(connection) }()

	sendDatagram(t, socket, `{"hud_type":"operation_finish","operation":"op1"}`)
	select {
	case <-shown:
	case <-time.After(2 * time.Second):
		t.Fatal("toast was never shown")
	}
	_ = connection.Close()
	select {
	case err := <-done:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("listen error = %v, want net.ErrClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("listen stayed blocked while a toast was being shown")
	}
}

func listenOnTempSocket(t *testing.T) (*net.UnixConn, string) {
	t.Helper()
	// Unix socket paths are capped near 100 bytes, which t.TempDir can exceed.
	dir, err := os.MkdirTemp("", "hud")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "events.sock")
	connection, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: socket, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	return connection, socket
}

func sendDatagram(t *testing.T, socket, payload string) {
	t.Helper()
	connection, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: socket, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = connection.Close() }()
	if _, err := connection.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
}

func recordingRunner(t *testing.T) (*runner, *[][2]string) {
	t.Helper()
	toasts := &[][2]string{}
	runner := &runner{
		pending: make(map[string]*pendingOperation),
		notify: func(title, body string) error {
			*toasts = append(*toasts, [2]string{title, body})
			return nil
		},
	}
	return runner, toasts
}
