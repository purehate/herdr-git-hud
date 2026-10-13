package operation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/purehate/herdr-git-hud/internal/gitinfo"
	"github.com/purehate/herdr-git-hud/internal/model"
	"github.com/purehate/herdr-git-hud/internal/traceevent"
)

type command struct {
	Label string
	Args  []string
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	parsed, err := parse(args)
	if err != nil {
		return 1, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return 1, fmt.Errorf("resolve working directory: %w", err)
	}
	repo := cwd
	if root, rootErr := gitinfo.Root(cwd); rootErr == nil {
		repo = root
	}

	operationID := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	send(traceevent.Event{
		HUDType:        "operation_start",
		Operation:      operationID,
		OperationLabel: parsed.Label,
		Repo:           traceevent.RepoField(repo),
	})

	child := exec.Command(parsed.Args[0], parsed.Args[1:]...)
	child.Stdin = stdin
	child.Stdout = stdout
	child.Stderr = stderr
	runErr := child.Run()
	exitCode := 0
	if runErr != nil {
		var exitError *exec.ExitError
		if !errors.As(runErr, &exitError) {
			sendFinish(operationID, parsed.Label, repo, 1)
			return 1, fmt.Errorf("run %s: %w", parsed.Args[0], runErr)
		}
		exitCode = exitError.ExitCode()
	}
	sendFinish(operationID, parsed.Label, repo, exitCode)
	return exitCode, nil
}

func parse(args []string) (command, error) {
	parsed := command{}
	for len(args) > 0 {
		switch args[0] {
		case "--":
			args = args[1:]
			parsed.Args = args
			if len(parsed.Args) == 0 {
				return command{}, fmt.Errorf("usage: herdr-git-hud run [--label TEXT] -- <command> [args...]")
			}
			if parsed.Label == "" {
				parsed.Label = strings.Join(parsed.Args, " ")
			}
			return parsed, nil
		case "--label":
			if len(args) < 2 || args[1] == "" {
				return command{}, fmt.Errorf("--label requires a value")
			}
			parsed.Label = args[1]
			args = args[2:]
		default:
			return command{}, fmt.Errorf("expected -- before command, got %q", args[0])
		}
	}
	return command{}, fmt.Errorf("usage: herdr-git-hud run [--label TEXT] -- <command> [args...]")
}

func sendFinish(operationID, label, repo string, exitCode int) {
	send(traceevent.Event{
		HUDType:        "operation_finish",
		Operation:      operationID,
		OperationLabel: label,
		Repo:           traceevent.RepoField(repo),
		ExitCode:       exitCode,
	})
}

func send(event traceevent.Event) {
	paths, err := model.PathsFromEnv()
	if err != nil {
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	connection, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: filepath.Clean(paths.Socket), Net: "unixgram"})
	if err != nil {
		return
	}
	defer func() { _ = connection.Close() }()
	_ = connection.SetWriteDeadline(time.Now().Add(250 * time.Millisecond))
	_, _ = connection.Write(payload)
}
