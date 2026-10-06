package daemon

import (
	"path/filepath"
	"testing"

	"github.com/purehate/herdr-git-hud/internal/model"
)

func TestFinishKeepsFailureVisible(t *testing.T) {
	runner, operation := finishingRunner(t)
	runner.finish(operation, 7)
	defer runner.closeTimer.Stop()

	if runner.state.Result != "failure" || runner.state.CloseAt != 0 {
		t.Fatalf("failure state = %#v, want failure with no automatic close", runner.state)
	}
	if runner.state.Steps[0].ExitCode != 7 {
		t.Fatalf("exit code = %d, want 7", runner.state.Steps[0].ExitCode)
	}
}

func TestFinishSchedulesSuccessfulClose(t *testing.T) {
	runner, operation := finishingRunner(t)
	runner.finish(operation, 0)
	defer runner.closeTimer.Stop()

	if runner.state.Result != "success" || runner.state.CloseAt <= model.NowMillis() {
		t.Fatalf("success state = %#v, want a future close time", runner.state)
	}
}

func finishingRunner(t *testing.T) (*runner, *pendingOperation) {
	t.Helper()
	stateFile := filepath.Join(t.TempDir(), "hud.json")
	operation := &pendingOperation{Visible: true, Step: 0, Sequence: 42, Repo: t.TempDir()}
	runner := &runner{
		paths: model.Paths{StateFile: stateFile},
		state: model.State{
			Sequence: 42,
			Result:   "running",
			Steps:    []model.Step{{Result: "running", StartedAt: model.NowMillis()}},
		},
		hudOpen: true,
	}
	return runner, operation
}
