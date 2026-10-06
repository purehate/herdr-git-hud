package control

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/purehate/herdr-git-hud/internal/daemon"
	"github.com/purehate/herdr-git-hud/internal/model"
)

const gitConfigKey = "trace2.eventTarget"

func Enable() error {
	paths, err := model.PathsFromEnv()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(paths.StateDir, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	target := "af_unix:dgram:" + paths.Socket
	current, found, err := readTarget()
	if err != nil {
		return err
	}
	if found && current != target {
		return fmt.Errorf("git already has %s=%q; refusing to overwrite it", gitConfigKey, current)
	}
	if err := gitConfig("--replace-all", gitConfigKey, target); err != nil {
		return err
	}
	if err := daemon.Ensure(); err != nil {
		return err
	}
	fmt.Println("Git HUD enabled")
	return nil
}

func Disable() error {
	paths, err := model.PathsFromEnv()
	if err != nil {
		return err
	}
	target := "af_unix:dgram:" + paths.Socket
	current, found, err := readTarget()
	if err != nil {
		return err
	}
	if found && current == target {
		if err := gitConfig("--unset-all", gitConfigKey); err != nil {
			return err
		}
	}
	if err := daemon.Stop(); err != nil {
		return err
	}
	fmt.Println("Git HUD disabled")
	return nil
}

func Status(writer io.Writer) error {
	paths, err := model.PathsFromEnv()
	if err != nil {
		return err
	}
	current, found, err := readTarget()
	if err != nil {
		return err
	}
	enabled := found && current == "af_unix:dgram:"+paths.Socket
	running := daemon.Running(paths.PIDFile)
	_, err = fmt.Fprintf(writer, "enabled: %t\ndaemon: %t\nsocket: %s\n", enabled, running, paths.Socket)
	return err
}

func readTarget() (string, bool, error) {
	command := gitCommand("config", "--global", "--get", gitConfigKey)
	output, err := command.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), true, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return "", false, nil
	}
	return "", false, fmt.Errorf("read Git Trace2 target: %w", err)
}

func gitConfig(args ...string) error {
	commandArgs := append([]string{"config", "--global"}, args...)
	command := gitCommand(commandArgs...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("configure Git Trace2: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func gitCommand(args ...string) *exec.Cmd {
	command := exec.Command("git", args...)
	command.Env = append(os.Environ(), "GIT_TRACE2_EVENT=0")
	return command
}
