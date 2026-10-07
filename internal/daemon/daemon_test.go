package daemon

import (
	"strings"
	"testing"

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
