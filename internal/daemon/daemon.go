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

type pendingOperation struct {
	SID     string
	Command string
	Label   string
	Argv    []string
	Repo    string
	Phase   string
	Visible bool
	Started int64
}

type runner struct {
	mu      sync.Mutex
	herdr   string
	pending map[string]*pendingOperation
	notify  func(title, body string) error
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

	r := &runner{herdr: herdrBinary(), pending: make(map[string]*pendingOperation)}
	r.notify = r.showToast
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
		if event.HUDType != "" {
			r.handleExternal(event)
		} else {
			events <- event
		}
	}
}

func (r *runner) handleExternal(event traceevent.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := "operation:" + event.Operation
	switch event.HUDType {
	case "operation_start":
		if event.Operation == "" || event.OperationLabel == "" || event.Repo == "" {
			return
		}
		operation := &pendingOperation{
			SID: key, Label: event.OperationLabel, Repo: filepath.Clean(event.Repo), Started: model.NowMillis(),
		}
		visible, err := r.repoIsVisible(operation.Repo)
		if err != nil || !visible {
			return
		}
		operation.Visible = true
		r.pending[key] = operation
	case "operation_finish":
		operation := r.pending[key]
		if operation == nil {
			return
		}
		r.finish(operation, event.ExitCode)
		delete(r.pending, key)
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
}

func (r *runner) updatePhase(operation *pendingOperation, phase string) {
	if operation.Visible && phase != "" {
		operation.Phase = phase
	}
}

func (r *runner) finish(operation *pendingOperation, code int) {
	if !operation.Visible {
		return
	}
	status, err := gitinfo.Collect(operation.Repo)
	if err != nil {
		log.Printf("collect status: %v", err)
	}
	head := ""
	if code == 0 && writesHistory[operation.Command] {
		if head, err = gitinfo.Head(operation.Repo); err != nil {
			log.Printf("read head: %v", err)
		}
	}
	elapsed := time.Duration(model.NowMillis()-operation.Started) * time.Millisecond
	title, body := toastText(operation, code, status, head, elapsed)
	if err := r.notify(title, body); err != nil {
		log.Printf("show toast: %v", err)
	}
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

// showToast uses Herdr's own notification toast, so placement follows
// [ui.toast.herdr] like agent completion notices and never takes focus.
func (r *runner) showToast(title, body string) error {
	_, err := r.herdrOutput("notification", "show", title, "--body", body, "--sound", "none")
	return err
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
