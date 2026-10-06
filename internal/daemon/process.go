package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/purehate/herdr-git-hud/internal/model"
)

func Command(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: herdr-git-hud daemon <ensure|run|stop>")
	}
	switch args[0] {
	case "ensure":
		return Ensure()
	case "run":
		return Run()
	case "stop":
		return Stop()
	default:
		return fmt.Errorf("unknown daemon command %q", args[0])
	}
}

func Ensure() error {
	paths, err := model.PathsFromEnv()
	if err != nil {
		return err
	}
	if Running(paths.PIDFile) {
		return nil
	}
	if err := os.MkdirAll(paths.StateDir, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	return spawn(paths)
}

func spawn(paths model.Paths) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	logFile, err := os.OpenFile(paths.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open daemon log: %w", err)
	}
	defer func() { _ = logFile.Close() }()
	command := exec.Command(executable, "daemon", "run")
	command.Stdin = nil
	command.Stdout = logFile
	command.Stderr = logFile
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}
	return command.Process.Release()
}

func Stop() error {
	paths, err := model.PathsFromEnv()
	if err != nil {
		return err
	}
	pid, err := readPID(paths.PIDFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find daemon process: %w", err)
	}
	if err := process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("stop daemon: %w", err)
	}
	return nil
}

func Running(pidFile string) bool {
	pid, err := readPID(pidFile)
	if err != nil {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func readPID(path string) (int, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(contents)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid daemon pid in %s", path)
	}
	return pid, nil
}
