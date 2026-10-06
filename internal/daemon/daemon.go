package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/purehate/herdr-git-hud/internal/gitinfo"
	"github.com/purehate/herdr-git-hud/internal/model"
	"github.com/purehate/herdr-git-hud/internal/traceevent"
)

const (
	lingerDuration = 2200 * time.Millisecond
	pluginID       = "purehate.git-hud"
)

type pendingOperation struct {
	SID      string
	Command  string
	Argv     []string
	Repo     string
	Visible  bool
	Step     int
	Started  int64
	Sequence int64
}

type runner struct {
	mu         sync.Mutex
	paths      model.Paths
	herdr      string
	pending    map[string]*pendingOperation
	state      model.State
	hudOpen    bool
	closeTimer *time.Timer
}

func Run() error {
	paths, err := model.PathsFromEnv()
	if err != nil {
		return err
	}
	if err := prepareSocket(paths); err != nil {
		return err
	}
	connection, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: paths.Socket, Net: "unixgram"})
	if err != nil {
		return fmt.Errorf("listen on Trace2 socket: %w", err)
	}
	defer func() { _ = connection.Close() }()
	defer func() { _ = os.Remove(paths.Socket) }()
	if err := os.WriteFile(paths.PIDFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o600); err != nil {
		return fmt.Errorf("write daemon pid: %w", err)
	}
	defer func() { _ = os.Remove(paths.PIDFile) }()

	r := &runner{paths: paths, herdr: herdrBinary(), pending: make(map[string]*pendingOperation)}
	defer r.closeHUD()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)
	go func() {
		<-stop
		_ = connection.Close()
	}()
	err = r.listen(connection)
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func prepareSocket(paths model.Paths) error {
	if err := os.MkdirAll(paths.StateDir, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	info, err := os.Lstat(paths.Socket)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect old Trace2 socket: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to replace non-socket path %s", paths.Socket)
	}
	if err := os.Remove(paths.Socket); err != nil {
		return fmt.Errorf("remove stale Trace2 socket: %w", err)
	}
	return nil
}

func (r *runner) listen(connection *net.UnixConn) error {
	events := make(chan traceevent.Event, 8192)
	go func() {
		for event := range events {
			r.handle(event)
		}
	}()

	buffer := make([]byte, 64*1024)
	for {
		n, _, err := connection.ReadFromUnix(buffer)
		if err != nil {
			return fmt.Errorf("read Trace2 event: %w", err)
		}
		var event traceevent.Event
		if err := json.Unmarshal(buffer[:n], &event); err != nil {
			continue
		}
		events <- event
	}
}

func (r *runner) handle(event traceevent.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	root := event.RootSID()
	operation := r.pending[root]
	if event.Event == "start" && event.IsRoot() {
		r.pending[root] = &pendingOperation{SID: root, Argv: event.Argv, Started: model.NowMillis()}
		return
	}
	if operation == nil {
		return
	}
	switch event.Event {
	case "def_repo":
		if operation.Repo == "" {
			operation.Repo = filepath.Clean(event.Worktree)
		}
	case "cmd_name":
		if event.IsRoot() {
			operation.Command = event.Name
			r.tryStart(operation)
		}
	case "child_start", "region_enter":
		r.updatePhase(operation, traceevent.Phase(event))
	case "exit":
		if event.IsRoot() {
			r.finish(operation, event.Code)
			delete(r.pending, root)
		}
	}
}

func (r *runner) tryStart(operation *pendingOperation) {
	if operation.Repo == "" || !traceevent.IsVisibleOperation(operation.Command, operation.Argv) {
		return
	}
	visible, err := r.repoIsVisible(operation.Repo)
	if err != nil || !visible {
		return
	}
	operation.Visible = true
	operation.Sequence = model.NowMillis()
	status, _ := gitinfo.Collect(operation.Repo)
	step := model.Step{Command: operation.Command, Phase: "Starting", Result: "running", StartedAt: operation.Started}
	if r.state.Result == "running" || model.NowMillis() < r.state.CloseAt {
		r.state.Steps = append(r.state.Steps, step)
	} else {
		r.state = model.State{StartedAt: operation.Started, Steps: []model.Step{step}}
	}
	operation.Step = len(r.state.Steps) - 1
	r.state.Sequence = operation.Sequence
	r.state.Repo = operation.Repo
	r.state.RepoName = filepath.Base(operation.Repo)
	r.state.Phase = "Starting git " + operation.Command
	r.state.Result = "running"
	r.state.CloseAt = 0
	r.state.Status = status
	if err := writeState(r.paths.StateFile, r.state); err != nil {
		log.Printf("write starting state: %v", err)
	}
	if r.closeTimer != nil {
		r.closeTimer.Stop()
	}
	if !r.hudOpen {
		if err := r.openHUD(operation.Repo); err != nil {
			log.Printf("open popup: %v", err)
		} else {
			r.hudOpen = true
		}
	}
}

func (r *runner) updatePhase(operation *pendingOperation, phase string) {
	if !operation.Visible || phase == "" || operation.Step >= len(r.state.Steps) {
		return
	}
	r.state.Phase = phase
	r.state.Steps[operation.Step].Phase = phase
	if err := writeState(r.paths.StateFile, r.state); err != nil {
		log.Printf("write phase state: %v", err)
	}
}

func (r *runner) finish(operation *pendingOperation, code int) {
	if !operation.Visible || operation.Step >= len(r.state.Steps) {
		return
	}
	result := "success"
	phase := "Completed"
	if code != 0 {
		result = "failure"
		phase = "Failed"
	}
	status, _ := gitinfo.Collect(operation.Repo)
	step := &r.state.Steps[operation.Step]
	step.Result = result
	step.ExitCode = code
	step.Phase = phase
	step.FinishedAt = model.NowMillis()
	r.state.Result = result
	r.state.Phase = phase
	r.state.Status = status
	r.state.CloseAt = time.Now().Add(lingerDuration).UnixMilli()
	if err := writeState(r.paths.StateFile, r.state); err != nil {
		log.Printf("write completion state: %v", err)
	}
	r.scheduleClose(operation.Sequence)
}

func (r *runner) closeHUD() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closeTimer != nil {
		r.closeTimer.Stop()
	}
	if r.hudOpen {
		r.state.CloseAt = model.NowMillis()
		_ = writeState(r.paths.StateFile, r.state)
		r.hudOpen = false
	}
}

func (r *runner) scheduleClose(sequence int64) {
	r.closeTimer = time.AfterFunc(lingerDuration, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.state.Sequence != sequence || r.state.Result == "running" {
			return
		}
		r.hudOpen = false
	})
}

func (r *runner) repoIsVisible(repo string) (bool, error) {
	output, err := r.herdrOutput("pane", "current")
	if err != nil {
		return false, err
	}
	var response struct {
		Result struct {
			Pane struct {
				CWD           string `json:"cwd"`
				ForegroundCWD string `json:"foreground_cwd"`
			} `json:"pane"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return false, fmt.Errorf("decode focused pane: %w", err)
	}
	cwd := response.Result.Pane.ForegroundCWD
	if cwd == "" {
		cwd = response.Result.Pane.CWD
	}
	activeRepo, err := gitinfo.Root(cwd)
	return err == nil && samePath(activeRepo, repo), err
}

func (r *runner) openHUD(repo string) error {
	_, err := r.herdrOutput("plugin", "pane", "open", "--plugin", pluginID,
		"--entrypoint", "hud", "--cwd", repo,
		"--env", "GIT_HUD_STATE_FILE="+r.paths.StateFile)
	if err != nil {
		return err
	}
	return nil
}

func (r *runner) herdrOutput(args ...string) ([]byte, error) {
	command := exec.Command(r.herdr, args...)
	command.Env = cleanHerdrEnv()
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("herdr %s: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

func cleanHerdrEnv() []string {
	blocked := []string{"HERDR_PANE_ID=", "HERDR_TAB_ID=", "HERDR_WORKSPACE_ID=", "GIT_TRACE2_EVENT="}
	environment := make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		keep := true
		for _, prefix := range blocked {
			keep = keep && !strings.HasPrefix(item, prefix)
		}
		if keep {
			environment = append(environment, item)
		}
	}
	return append(environment, "GIT_TRACE2_EVENT=0")
}

func herdrBinary() string {
	if value := os.Getenv("HERDR_BIN_PATH"); value != "" {
		return value
	}
	return "herdr"
}

func samePath(left, right string) bool {
	leftResolved, leftErr := filepath.EvalSymlinks(left)
	rightResolved, rightErr := filepath.EvalSymlinks(right)
	if leftErr == nil && rightErr == nil {
		return leftResolved == rightResolved
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func writeState(path string, state model.State) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".hud-*.json")
	if err != nil {
		return fmt.Errorf("create HUD state: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	encoder := json.NewEncoder(temporary)
	if err := encoder.Encode(state); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("encode HUD state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close HUD state: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish HUD state: %w", err)
	}
	return nil
}
